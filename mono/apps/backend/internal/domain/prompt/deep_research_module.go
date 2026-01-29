package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"

	"jan-server/mono/apps/backend/internal/domain/modelprompttemplate"
	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
)

const (
	deepResearchModuleName = "deep_research"
)

// DeepResearchModule injects the Deep Research system prompt when enabled
type DeepResearchModule struct {
	templateService      *prompttemplate.Service
	modelPromptService   *modelprompttemplate.Service
}

// NewDeepResearchModule creates a new deep research module
func NewDeepResearchModule(templateService *prompttemplate.Service) *DeepResearchModule {
	return &DeepResearchModule{
		templateService: templateService,
	}
}

// NewDeepResearchModuleWithModelPrompts creates a new deep research module with model-specific prompt support
func NewDeepResearchModuleWithModelPrompts(
	templateService *prompttemplate.Service,
	modelPromptService *modelprompttemplate.Service,
) *DeepResearchModule {
	return &DeepResearchModule{
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier
func (m *DeepResearchModule) Name() string {
	return deepResearchModuleName
}

// ShouldApply determines if the Deep Research prompt should be injected
// This module applies when:
// 1. Deep research is explicitly enabled in preferences
// 2. Module is not disabled
func (m *DeepResearchModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		log.Debug().Msg("[DEBUG] DeepResearchModule.ShouldApply: context is nil or cancelled")
		return false
	}
	if promptCtx == nil || promptCtx.Preferences == nil {
		log.Debug().Msg("[DEBUG] DeepResearchModule.ShouldApply: promptCtx or Preferences is nil")
		return false
	}

	// Check if module is disabled
	if isModuleDisabled(promptCtx.Preferences, m.Name()) {
		log.Debug().Msg("[DEBUG] DeepResearchModule.ShouldApply: module is disabled via preferences")
		return false
	}

	// Check if deep_research is enabled in preferences
	deepResearch, ok := promptCtx.Preferences["deep_research"]
	if !ok {
		log.Debug().
			Interface("preferences", promptCtx.Preferences).
			Msg("[DEBUG] DeepResearchModule.ShouldApply: deep_research not found in preferences")
		return false
	}

	log.Debug().
		Interface("deep_research_value", deepResearch).
		Str("deep_research_type", fmt.Sprintf("%T", deepResearch)).
		Msg("[DEBUG] DeepResearchModule.ShouldApply: found deep_research in preferences")

	// Handle different types
	switch v := deepResearch.(type) {
	case bool:
		log.Debug().Bool("result", v).Msg("[DEBUG] DeepResearchModule.ShouldApply: returning bool value")
		return v
	case string:
		result := strings.ToLower(v) == "true"
		log.Debug().Bool("result", result).Msg("[DEBUG] DeepResearchModule.ShouldApply: returning parsed string value")
		return result
	default:
		log.Debug().
			Str("type", fmt.Sprintf("%T", deepResearch)).
			Msg("[DEBUG] DeepResearchModule.ShouldApply: unsupported type, returning false")
		return false
	}
}

// Apply injects the Deep Research system prompt at the beginning of the messages
func (m *DeepResearchModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}

	var promptContent string
	var templateSource string

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("DeepResearchModule: Attempting to load model-specific template")
		
		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyDeepResearch)
		if err == nil && template != nil && template.IsActive {
			promptContent = template.Content
			templateSource = source
			log.Debug().
				Str("template_key", template.TemplateKey).
				Str("template_name", template.Name).
				Str("source", source).
				Str("model_catalog_id", *promptCtx.ModelCatalogID).
				Int("content_length", len(promptContent)).
				Msg("DeepResearchModule: Loaded prompt template")
		}
	}

	// Fall back to regular template service if model-specific not found
	if promptContent == "" && m.templateService != nil {
		log.Debug().Msg("DeepResearchModule: Attempting to load template from database")
		template, err := m.templateService.GetDeepResearchPrompt(ctx)
		if err == nil && template != nil && template.IsActive {
			promptContent = template.Content
			templateSource = "global_default"
			log.Debug().
				Str("template_key", template.TemplateKey).
				Str("template_name", template.Name).
				Str("source", templateSource).
				Int("content_length", len(promptContent)).
				Msg("DeepResearchModule: Loaded prompt template from database")
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("DeepResearchModule: Failed to load template from database, using fallback")
			} else {
				log.Warn().Msg("DeepResearchModule: Template is inactive or nil, using fallback")
			}
		}
	} else if promptContent == "" {
		log.Warn().Msg("DeepResearchModule: templateService is nil, using fallback")
	}

	// Fallback to the default constant if no template was found
	if promptContent == "" {
		promptContent = prompttemplate.DefaultDeepResearchPrompt
		templateSource = "hardcoded"
		log.Debug().
			Int("fallback_content_length", len(promptContent)).
			Str("source", templateSource).
			Msg("DeepResearchModule: Using fallback default prompt")
	}

	// Prepend the Deep Research system prompt
	return prependDeepResearchPrompt(messages, promptContent), nil
}

// prependDeepResearchPrompt prepends the deep research system prompt to messages
func prependDeepResearchPrompt(messages []openai.ChatCompletionMessage, content string) []openai.ChatCompletionMessage {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return messages
	}

	result := make([]openai.ChatCompletionMessage, 0, len(messages)+1)

	// Add Deep Research system message first
	result = append(result, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: trimmed,
	})

	// Append all existing messages
	for _, msg := range messages {
		result = append(result, cloneMessage(msg))
	}

	return result
}

// DeepResearchConfig holds configuration extracted from template metadata
type DeepResearchConfig struct {
	MaxTokens            int      `json:"max_tokens,omitempty"`
	Temperature          float32  `json:"temperature,omitempty"`
	RequiresTools        bool     `json:"requires_tools,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
}

// GetDeepResearchConfig extracts configuration from template metadata
func GetDeepResearchConfig(metadata map[string]any) *DeepResearchConfig {
	if metadata == nil {
		return &DeepResearchConfig{
			MaxTokens:            16384,
			Temperature:          0.7,
			RequiresTools:        true,
			RequiredCapabilities: []string{"reasoning"},
		}
	}

	config := &DeepResearchConfig{
		MaxTokens:            16384,
		Temperature:          0.7,
		RequiresTools:        true,
		RequiredCapabilities: []string{"reasoning"},
	}

	if maxTokens, ok := metadata["max_tokens"].(float64); ok {
		config.MaxTokens = int(maxTokens)
	}
	if temperature, ok := metadata["temperature"].(float64); ok {
		config.Temperature = float32(temperature)
	}
	if requiresTools, ok := metadata["requires_tools"].(bool); ok {
		config.RequiresTools = requiresTools
	}
	if capabilities, ok := metadata["required_capabilities"].([]interface{}); ok {
		config.RequiredCapabilities = make([]string, 0, len(capabilities))
		for _, cap := range capabilities {
			if s, ok := cap.(string); ok {
				config.RequiredCapabilities = append(config.RequiredCapabilities, s)
			}
		}
	}

	return config
}
