package prompt

import (
	"context"
	"fmt"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"

	"jan-server/services/llm-api/internal/domain/modelprompttemplate"
	"jan-server/services/llm-api/internal/domain/prompttemplate"
	"jan-server/services/llm-api/internal/domain/usersettings"
)

const (
	projectInstructionModuleName = "project_instruction"
	userProfileModuleName        = "user_profile"
)

func cloneMessage(msg openai.ChatCompletionMessage) openai.ChatCompletionMessage {
	clone := msg

	if len(msg.MultiContent) > 0 {
		clone.MultiContent = make([]openai.ChatMessagePart, len(msg.MultiContent))
		for i, part := range msg.MultiContent {
			clone.MultiContent[i] = part
			if part.ImageURL != nil {
				img := *part.ImageURL
				clone.MultiContent[i].ImageURL = &img
			}
		}
	}

	if len(msg.ToolCalls) > 0 {
		clone.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
		copy(clone.ToolCalls, msg.ToolCalls)
	}

	if msg.FunctionCall != nil {
		fn := *msg.FunctionCall
		clone.FunctionCall = &fn
	}

	return clone
}

// prependInstructionSystemMessage returns a copy of messages with the instruction system
// message prepended.
func prependInstructionSystemMessage(messages []openai.ChatCompletionMessage, instruction, moduleName string) []openai.ChatCompletionMessage {
	trimmed := strings.TrimSpace(instruction)
	if trimmed == "" {
		return messages
	}

	var builder strings.Builder
	builder.WriteString(trimmed)

	// Special handling for project instructions: explicitly state priority
	if moduleName == projectInstructionModuleName {
		builder.WriteString("\n\n")
		builder.WriteString("Project priority: These project-specific instructions have the highest priority. ")
		builder.WriteString("If any user settings, style preferences, or other guidance conflict with these project instructions, ")
		builder.WriteString("you must follow the project instructions.")
	}

	result := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	result = append(result, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: builder.String(),
	})

	for _, msg := range messages {
		result = append(result, cloneMessage(msg))
	}

	return result
}

// PrependProjectInstruction injects the project instruction as the first system message.
func PrependProjectInstruction(messages []openai.ChatCompletionMessage, instruction string) []openai.ChatCompletionMessage {
	return prependInstructionSystemMessage(messages, instruction, projectInstructionModuleName)
}

// appendSystemContent attaches "additional" instructions into a suitable system message
// or creates a new system message if needed.
//
// Rules:
//   - We NEVER modify the first system message if it contains project instructions.
//   - When we create a new system message, we insert it right after the first
//     system message (if present) so project instructions stay topologically first.
//   - baseContent is the base system content (if non-empty) when creating a new message.
func appendSystemContent(
	messages []openai.ChatCompletionMessage,
	baseContent, moduleName, additional string,
) []openai.ChatCompletionMessage {
	additional = strings.TrimSpace(additional)

	log.Debug().
		Str("module", moduleName).
		Int("base_content_length", len(baseContent)).
		Int("additional_length", len(additional)).
		Int("input_messages_count", len(messages)).
		Msg("appendSystemContent: called")

	result := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	applied := false
	firstSystemIdx := -1

	for i, m := range messages {
		msg := cloneMessage(m)
		if msg.Role == openai.ChatMessageRoleSystem {
			if firstSystemIdx == -1 {
				firstSystemIdx = i
				log.Debug().
					Str("module", moduleName).
					Int("first_system_idx", i).
					Int("content_length", len(msg.Content)).
					Msg("appendSystemContent: found first system message (not modifying)")
			} else if !applied && additional != "" {
				// Append to subsequent system messages
				var b strings.Builder
				b.WriteString(strings.TrimSpace(msg.Content))
				b.WriteString("\n\n")
				b.WriteString(additional)
				msg.Content = b.String()
				applied = true
				log.Debug().
					Str("module", moduleName).
					Int("system_msg_idx", i).
					Msg("appendSystemContent: appended additional to existing system message")
			}
		}
		result = append(result, msg)
	}

	// If we successfully appended to an existing system message, we're done.
	if applied {
		log.Debug().
			Str("module", moduleName).
			Msg("appendSystemContent: done (appended to existing)")
		return result
	}

	// If we have nothing to say and no base content, just return.
	if additional == "" && strings.TrimSpace(baseContent) == "" {
		log.Debug().
			Str("module", moduleName).
			Msg("appendSystemContent: nothing to add, returning unchanged")
		return result
	}

	// Create a new system message with base content + additional content.
	var builder strings.Builder
	baseText := strings.TrimSpace(baseContent)
	if baseText == "" {
		baseText = "You are Jan, a helpful AI assistant.  Jan is trained by Menlo Research (https://www.menlo.ai) - a fame research lab. "
		log.Warn().
			Str("module", moduleName).
			Msg("appendSystemContent: baseContent was empty, using hardcoded fallback!")
	}
	builder.WriteString(baseText)

	if additional != "" {
		builder.WriteString("\n\n")
		builder.WriteString(additional)
	}

	finalContent := builder.String()
	log.Debug().
		Str("module", moduleName).
		Int("final_content_length", len(finalContent)).
		Str("final_content_preview", truncateString(finalContent, 200)).
		Msg("appendSystemContent: creating new system message")

	systemMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: finalContent,
	}

	// Insert AFTER first system message if it exists,
	// so project instructions stay top priority.
	insertIdx := 0
	if firstSystemIdx >= 0 {
		insertIdx = firstSystemIdx + 1
	}

	log.Debug().
		Str("module", moduleName).
		Int("insert_idx", insertIdx).
		Int("first_system_idx", firstSystemIdx).
		Msg("appendSystemContent: inserting new system message")

	// Insert at insertIdx
	result = append(result, openai.ChatCompletionMessage{}) // grow slice
	copy(result[insertIdx+1:], result[insertIdx:])
	result[insertIdx] = systemMsg

	return result
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func disabledModules(preferences map[string]interface{}) map[string]struct{} {
	disabled := map[string]struct{}{}
	if preferences == nil {
		return disabled
	}
	raw, ok := preferences["disable_modules"]
	if !ok {
		return disabled
	}
	switch v := raw.(type) {
	case string:
		for _, part := range strings.Split(v, ",") {
			if trimmed := strings.ToLower(strings.TrimSpace(part)); trimmed != "" {
				disabled[trimmed] = struct{}{}
			}
		}
	case []string:
		for _, part := range v {
			if trimmed := strings.ToLower(strings.TrimSpace(part)); trimmed != "" {
				disabled[trimmed] = struct{}{}
			}
		}
	case []interface{}:
		for _, part := range v {
			if str, ok := part.(string); ok {
				if trimmed := strings.ToLower(strings.TrimSpace(str)); trimmed != "" {
					disabled[trimmed] = struct{}{}
				}
			}
		}
	}
	return disabled
}

func isModuleDisabled(preferences map[string]interface{}, moduleName string) bool {
	disabled := disabledModules(preferences)
	_, found := disabled[strings.ToLower(moduleName)]
	return found
}

// ProjectInstructionModule injects project-specific instructions at the start of the conversation.
type ProjectInstructionModule struct{}

// NewProjectInstructionModule creates a new project instruction module.
func NewProjectInstructionModule() *ProjectInstructionModule {
	return &ProjectInstructionModule{}
}

// Name returns the module identifier.
func (m *ProjectInstructionModule) Name() string {
	return projectInstructionModuleName
}

// ShouldApply determines if project instructions should be injected.
func (m *ProjectInstructionModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx == nil {
		return false
	}
	if promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	return strings.TrimSpace(promptCtx.ProjectInstruction) != ""
}

// Apply prepends the project instruction as a system message.
func (m *ProjectInstructionModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}
	if promptCtx == nil || strings.TrimSpace(promptCtx.ProjectInstruction) == "" {
		return messages, nil
	}

	return PrependProjectInstruction(messages, promptCtx.ProjectInstruction), nil
}

// TimingModule injects the AI assistant intro and current date into the system prompt.
type TimingModule struct {
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewTimingModule creates a new timing module without template service (uses fallback).
func NewTimingModule() *TimingModule {
	return &TimingModule{}
}

// NewTimingModuleWithService creates a new timing module with template service.
func NewTimingModuleWithService(service *prompttemplate.Service) *TimingModule {
	return &TimingModule{
		templateService: service,
	}
}

// NewTimingModuleWithModelPrompts creates a new timing module with model-specific template support.
func NewTimingModuleWithModelPrompts(templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *TimingModule {
	return &TimingModule{
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *TimingModule) Name() string {
	return "timing"
}

// ShouldApply always applies when prompt orchestration is enabled and module not disabled.
func (m *TimingModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx != nil && promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	return true
}

// Apply injects the AI assistant intro and current date.
func (m *TimingModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}

	// Format current date as "Month Day, Year" (e.g., "November 28, 2025")
	currentDate := time.Now().Format("January 2, 2006")

	var timingText string
	var templateSource string

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("TimingModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyTiming)
		if err == nil && template != nil && template.IsActive {
			// Render template with current date variable
			rendered, renderErr := renderTemplateContent(template.Content, map[string]any{
				"CurrentDate": currentDate,
			})
			if renderErr == nil {
				timingText = rendered
				templateSource = source
				log.Debug().
					Str("module", "timing").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Str("current_date", currentDate).
					Int("content_length", len(timingText)).
					Int("template_version", template.Version).
					Msg("TimingModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "timing").
					Str("rendered_content", timingText).
					Msg("TimingModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("TimingModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if timingText == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyTiming)
		if err == nil && template != nil && template.IsActive {
			// Render template with current date variable
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyTiming, map[string]any{
				"CurrentDate": currentDate,
			})
			if renderErr == nil {
				timingText = rendered
				templateSource = "global_default"
				log.Debug().
					Str("module", "timing").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", templateSource).
					Str("current_date", currentDate).
					Int("content_length", len(timingText)).
					Int("template_version", template.Version).
					Msg("TimingModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "timing").
					Str("rendered_content", timingText).
					Msg("TimingModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("TimingModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("TimingModule: Failed to load template from database, using fallback")
			}
		}
	}

	_ = templateSource // used in logging

	// Fallback to hardcoded timing text if template not loaded
	if timingText == "" {
		timingText = fmt.Sprintf(
			"You are Jan, a helpful AI assistant who helps the user with their requests. Jan is trained by Menlo Research (https://www.menlo.ai) - a fame research lab.\n"+
				"Today is: %s.\n"+
				"Always treat this as the current date.",
			currentDate,
		)
		log.Debug().
			Str("module", "timing").
			Str("source", "hardcoded_fallback").
			Int("content_length", len(timingText)).
			Msg("TimingModule: Using fallback hardcoded prompt")
	}

	result := appendSystemContent(messages, timingText, m.Name(), "")
	return result, nil
}

// UserProfileModule injects user profile personalization into the system prompt.
type UserProfileModule struct {
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewUserProfileModule creates a new user profile module without template service (uses fallback).
func NewUserProfileModule() *UserProfileModule {
	return &UserProfileModule{}
}

// NewUserProfileModuleWithService creates a new user profile module with template service.
func NewUserProfileModuleWithService(service *prompttemplate.Service) *UserProfileModule {
	return &UserProfileModule{
		templateService: service,
	}
}

// NewUserProfileModuleWithModelPrompts creates a new user profile module with model-specific template support.
func NewUserProfileModuleWithModelPrompts(templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *UserProfileModule {
	return &UserProfileModule{
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *UserProfileModule) Name() string {
	return userProfileModuleName
}

// ShouldApply determines if user profile information should be injected.
func (m *UserProfileModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx == nil || promptCtx.Profile == nil {
		return false
	}
	if promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}

	profile := promptCtx.Profile

	// Apply when any personalization field is present (base style defaults to Friendly so non-empty).
	return profile.BaseStyle != "" ||
		strings.TrimSpace(profile.CustomInstructions) != "" ||
		strings.TrimSpace(profile.NickName) != "" ||
		strings.TrimSpace(profile.Occupation) != "" ||
		strings.TrimSpace(profile.MoreAboutYou) != ""
}

func baseStyleInstruction(style usersettings.BaseStyle) string {
	switch style {
	case usersettings.BaseStyleConcise:
		return "Use a concise style: brief, direct answers with minimal filler."
	case usersettings.BaseStyleFriendly:
		return "Use a friendly, warm, and encouraging tone while staying helpful."
	case usersettings.BaseStyleProfessional:
		return "Use a professional, clear, and structured tone appropriate for business settings."
	default:
		if strings.TrimSpace(string(style)) != "" {
			return fmt.Sprintf("Use the user's preferred style: %s.", style)
		}
		return ""
	}
}

// Apply injects user profile guidance and persona instructions.
func (m *UserProfileModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}
	if promptCtx == nil || promptCtx.Profile == nil {
		return messages, nil
	}

	var instruction string
	var templateSource string

	// Build variables for template rendering
	vars := map[string]any{
		"BaseStyle":          string(promptCtx.Profile.BaseStyle),
		"CustomInstructions": promptCtx.Profile.CustomInstructions,
		"NickName":           promptCtx.Profile.NickName,
		"Occupation":         promptCtx.Profile.Occupation,
		"MoreAboutYou":       promptCtx.Profile.MoreAboutYou,
	}

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("UserProfileModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyUserProfile)
		if err == nil && template != nil && template.IsActive {
			// Render template with user profile variables
			rendered, renderErr := renderTemplateContent(template.Content, vars)
			if renderErr == nil {
				instruction = rendered
				templateSource = source
				log.Debug().
					Str("module", "user_profile").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Int("content_length", len(instruction)).
					Int("template_version", template.Version).
					Msg("UserProfileModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "user_profile").
					Str("rendered_content", instruction).
					Msg("UserProfileModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("UserProfileModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if instruction == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyUserProfile)
		if err == nil && template != nil && template.IsActive {
			// Render template with user profile variables
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyUserProfile, vars)
			if renderErr == nil {
				instruction = rendered
				templateSource = "global_default"
				log.Debug().
					Str("module", "user_profile").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", templateSource).
					Int("content_length", len(instruction)).
					Int("template_version", template.Version).
					Msg("UserProfileModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "user_profile").
					Str("rendered_content", instruction).
					Msg("UserProfileModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("UserProfileModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("UserProfileModule: Failed to load template from database, using fallback")
			}
		}
	}

	_ = templateSource // used in logging

	// Fallback to hardcoded logic if template not loaded
	if instruction == "" {
		profile := promptCtx.Profile
		var sections []string

		// General note about precedence vs project instructions.
		sections = append(sections,
			"User-level settings are preferences for style and context. "+
				"If they ever conflict with explicit project or system instructions, always follow the project or system instructions.")

		if styleText := baseStyleInstruction(profile.BaseStyle); styleText != "" {
			sections = append(sections, styleText)
		}

		if custom := strings.TrimSpace(profile.CustomInstructions); custom != "" {
			sections = append(sections, fmt.Sprintf("Custom instructions from the user:\n%s", custom))
		}

		var details []string
		if nick := strings.TrimSpace(profile.NickName); nick != "" {
			details = append(details, fmt.Sprintf("Address the user as \"%s\".", nick))
		}
		if occupation := strings.TrimSpace(profile.Occupation); occupation != "" {
			details = append(details, fmt.Sprintf("Occupation: %s.", occupation))
		}
		if more := strings.TrimSpace(profile.MoreAboutYou); more != "" {
			details = append(details, fmt.Sprintf("About the user: %s.", more))
		}
		if len(details) > 0 {
			var builder strings.Builder
			builder.WriteString("User context:\n")
			for _, detail := range details {
				builder.WriteString("- ")
				builder.WriteString(detail)
				builder.WriteString("\n")
			}
			sections = append(sections, strings.TrimSpace(builder.String()))
		}

		instruction = strings.TrimSpace(strings.Join(sections, "\n\n"))
		log.Debug().Msg("UserProfileModule: Using fallback hardcoded prompt")
	}

	if instruction == "" {
		return messages, nil
	}

	result := appendSystemContent(messages, instruction, m.Name(), "")
	return result, nil
}

// WithDisabledModules returns a shallow copy of Context with module disable list merged.
func WithDisabledModules(ctx *Context, disable []string) *Context {
	if ctx == nil {
		return &Context{
			Preferences: map[string]interface{}{
				"disable_modules": disable,
			},
		}
	}
	prefs := ctx.Preferences
	if prefs == nil {
		prefs = map[string]interface{}{}
	}
	prefs["disable_modules"] = disable
	ctx.Preferences = prefs
	return ctx
}

// MemoryModule adds user memory to system prompts.
type MemoryModule struct {
	enabled            bool
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewMemoryModule creates a new memory module without template service (uses fallback).
func NewMemoryModule(enabled bool) *MemoryModule {
	return &MemoryModule{enabled: enabled}
}

// NewMemoryModuleWithService creates a new memory module with template service.
func NewMemoryModuleWithService(enabled bool, service *prompttemplate.Service) *MemoryModule {
	return &MemoryModule{
		enabled:         enabled,
		templateService: service,
	}
}

// NewMemoryModuleWithModelPrompts creates a new memory module with model-specific template support.
func NewMemoryModuleWithModelPrompts(enabled bool, templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *MemoryModule {
	return &MemoryModule{
		enabled:            enabled,
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *MemoryModule) Name() string {
	return "memory"
}

// ShouldApply checks if memory should be included.
func (m *MemoryModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if !m.enabled || promptCtx == nil {
		return false
	}
	if promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	return len(promptCtx.Memory) > 0
}

// Apply adds memory to the system prompt.
func (m *MemoryModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}
	if promptCtx == nil || len(promptCtx.Memory) == 0 {
		return messages, nil
	}

	var memoryText string

	// Build memory items for template rendering
	vars := map[string]any{
		"MemoryItems": promptCtx.Memory,
	}

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("MemoryModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyMemory)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := renderTemplateContent(template.Content, vars)
			if renderErr == nil {
				memoryText = rendered
				log.Debug().
					Str("module", "memory").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Int("memory_count", len(promptCtx.Memory)).
					Msg("MemoryModule: Loaded and rendered template from database")
			} else {
				log.Warn().Err(renderErr).Msg("MemoryModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if memoryText == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyMemory)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyMemory, vars)
			if renderErr == nil {
				memoryText = rendered
				log.Debug().
					Str("module", "memory").
					Str("template_key", template.TemplateKey).
					Str("source", "global_default").
					Int("memory_count", len(promptCtx.Memory)).
					Msg("MemoryModule: Loaded and rendered template from database")
			} else {
				log.Warn().Err(renderErr).Msg("MemoryModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("MemoryModule: Failed to load template from database, using fallback")
			}
		}
	}

	// Fallback to hardcoded memory text if template not loaded
	if memoryText == "" {
		var builder strings.Builder
		builder.WriteString("Use the following personal memory for this user when helpful, without overriding project or system instructions:\n")
		for _, item := range promptCtx.Memory {
			builder.WriteString("- ")
			builder.WriteString(item)
			builder.WriteString("\n")
		}
		memoryText = strings.TrimSpace(builder.String())
		log.Debug().Msg("MemoryModule: Using fallback hardcoded prompt")
	}

	result := appendSystemContent(messages, memoryText, m.Name(), "")
	return result, nil
}

// ToolInstructionsModule adds tool usage instructions.
type ToolInstructionsModule struct {
	enabled            bool
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewToolInstructionsModule creates a new tool instructions module without template service (uses fallback).
func NewToolInstructionsModule(enabled bool) *ToolInstructionsModule {
	return &ToolInstructionsModule{enabled: enabled}
}

// NewToolInstructionsModuleWithService creates a new tool instructions module with template service.
func NewToolInstructionsModuleWithService(enabled bool, service *prompttemplate.Service) *ToolInstructionsModule {
	return &ToolInstructionsModule{
		enabled:         enabled,
		templateService: service,
	}
}

// NewToolInstructionsModuleWithModelPrompts creates a new tool instructions module with model-specific template support.
func NewToolInstructionsModuleWithModelPrompts(enabled bool, templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *ToolInstructionsModule {
	return &ToolInstructionsModule{
		enabled:            enabled,
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *ToolInstructionsModule) Name() string {
	return "tool_instructions"
}

// ShouldApply checks if tool instructions should be added.
func (m *ToolInstructionsModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if !m.enabled {
		return false
	}
	if promptCtx == nil {
		return false
	}
	if promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}

	return detectToolUsage(promptCtx, messages)
}

// Apply adds tool instructions to the system prompt.
func (m *ToolInstructionsModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}
	if promptCtx == nil || !detectToolUsage(promptCtx, messages) {
		return messages, nil
	}

	var toolText string

	// Build variables for template rendering
	vars := buildToolTemplateVars(promptCtx)

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("ToolInstructionsModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyToolInstructions)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := renderTemplateContent(template.Content, vars)
			if renderErr == nil {
				toolText = rendered
				log.Debug().
					Str("module", "tool_instructions").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("ToolInstructionsModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "tool_instructions").
					Str("rendered_content", rendered).
					Msg("ToolInstructionsModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("ToolInstructionsModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if toolText == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyToolInstructions)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyToolInstructions, vars)
			if renderErr == nil {
				toolText = rendered
				log.Debug().
					Str("module", "tool_instructions").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", "global_default").
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("ToolInstructionsModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "tool_instructions").
					Str("rendered_content", rendered).
					Msg("ToolInstructionsModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("ToolInstructionsModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("ToolInstructionsModule: Failed to load template from database, using fallback")
			}
		}
	}

	// Fallback to hardcoded tool text if template not loaded
	if toolText == "" {
		// Use the same vars that would be used for template rendering
		if len(vars["Tools"].([]map[string]string)) == 0 {
			// No tools available at all
			log.Debug().Msg("ToolInstructionsModule: No tools available, skipping")
			return messages, nil
		}

		var builder strings.Builder
		builder.WriteString("## Tool Usage Instructions\n\n")
		builder.WriteString("You have access to the following tools. **ONLY use tools from this list. Do not invent or claim access to tools not listed here.**\n\n")
		builder.WriteString("AVAILABLE TOOLS:\n")

		// Build explicit tool list from parsed tools
		tools := vars["Tools"].([]map[string]string)
		for _, tool := range tools {
			builder.WriteString("- **")
			builder.WriteString(tool["Name"])
			builder.WriteString("**: ")
			builder.WriteString(tool["Description"])
			if tool["Parameters"] != "" {
				builder.WriteString("\n  - Parameters: ")
				builder.WriteString(tool["Parameters"])
			}
			builder.WriteString("\n")
		}

		builder.WriteString("\nCRITICAL RULES:\n")
		builder.WriteString("1. **Only use tools from the list above** - Never claim access to tools not in this list\n")
		builder.WriteString("2. **If a tool is not listed, it does not exist** - Do not invent tool names or capabilities\n")
		builder.WriteString("3. **When asked about available tools**, list ONLY the tools from the list above\n")
		builder.WriteString("4. Always choose the best tool for the task from the available tools\n")
		builder.WriteString("5. Tool usage must respect project instructions and system-level constraints at all times\n")

		// Add usage patterns only for enabled tools
		builder.WriteString("\nTOOL USAGE PATTERNS:\n")
		if vars["HasSearchTool"].(bool) {
			builder.WriteString("- When you need to search for information: use ")
			builder.WriteString(vars["SearchToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasScrapeTool"].(bool) {
			builder.WriteString("- When you need to scrape or extract content from a webpage: use ")
			builder.WriteString(vars["ScrapeToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasCodeTool"].(bool) {
			builder.WriteString("- When you need to execute code: use ")
			builder.WriteString(vars["CodeToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasBrowserTool"].(bool) {
			builder.WriteString("- When you need to browse the web: use ")
			builder.WriteString(vars["BrowserToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasImageGenerateTool"].(bool) {
			builder.WriteString("- When you need to generate NEW images: use ")
			builder.WriteString(vars["ImageGenerateToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasImageEditTool"].(bool) {
			builder.WriteString("- When you need to edit EXISTING images: use ")
			builder.WriteString(vars["ImageEditToolName"].(string))
			builder.WriteString("\n")
		}
		if vars["HasImageTool"].(bool) {
			builder.WriteString("\nIMAGE OUTPUT FORMATTING:\n")
			builder.WriteString("- Always wrap generated or edited images in <img> tags, NOT <a> tags\n")
			builder.WriteString("- Example: <img src=\"image_url\" alt=\"description\" />\n")
			builder.WriteString("- Do NOT use: <a href=\"image_url\">...</a>\n")
		}

		toolText = strings.TrimSpace(builder.String())
		log.Debug().Msg("ToolInstructionsModule: Using fallback hardcoded prompt")
	}

	result := appendSystemContent(messages, toolText, m.Name(), "")
	return result, nil
}

// buildToolTemplateVars builds template variables for tool instructions
func buildToolTemplateVars(promptCtx *Context) map[string]any {
	vars := map[string]any{
		"Tools":                 []map[string]string{},
		"HasSearchTool":         false,
		"SearchToolName":        "",
		"HasCodeTool":           false,
		"CodeToolName":          "",
		"HasBrowserTool":        false,
		"BrowserToolName":       "",
		"HasScrapeTool":         false,
		"ScrapeToolName":        "",
		"HasImageTool":          false,
		"HasImageGenerateTool":  false,
		"ImageGenerateToolName": "",
		"HasImageEditTool":      false,
		"ImageEditToolName":     "",
	}

	if promptCtx == nil || len(promptCtx.Tools) == 0 {
		return vars
	}

	// Parse tools directly from request.Tools (OpenAI format)
	tools := make([]map[string]string, 0, len(promptCtx.Tools))

	for _, tool := range promptCtx.Tools {
		if tool.Type != openai.ToolTypeFunction {
			continue
		}

		toolName := tool.Function.Name
		toolDesc := tool.Function.Description

		// Extract parameters description
		paramsDesc := ""
		if tool.Function.Parameters != nil {
			// Type assert Parameters to map first
			if paramsMap, ok := tool.Function.Parameters.(map[string]interface{}); ok {
				// Try to extract required parameters
				if props, ok := paramsMap["properties"].(map[string]interface{}); ok {
					paramNames := []string{}
					for paramName := range props {
						paramNames = append(paramNames, paramName)
					}
					if len(paramNames) > 0 {
						paramsDesc = strings.Join(paramNames, ", ")
					}
				}
			}
		}

		tools = append(tools, map[string]string{
			"Name":        toolName,
			"Description": toolDesc,
			"Parameters":  paramsDesc,
		})

		// Detect tool types based on name and description
		toolNameLower := strings.ToLower(toolName)
		toolDescLower := strings.ToLower(toolDesc)

		// Search tools (web search, google search, etc.)
		if strings.Contains(toolNameLower, "search") ||
			strings.Contains(toolNameLower, "google") ||
			strings.Contains(toolNameLower, "web_search") ||
			strings.Contains(toolDescLower, "search") ||
			strings.Contains(toolDescLower, "web search") {
			vars["HasSearchTool"] = true
			vars["SearchToolName"] = toolName
		}

		// Code execution tools (python, code, execute, etc.)
		if strings.Contains(toolNameLower, "code") ||
			strings.Contains(toolNameLower, "execute") ||
			strings.Contains(toolNameLower, "python") ||
			strings.Contains(toolNameLower, "run_code") ||
			strings.Contains(toolDescLower, "execute code") ||
			strings.Contains(toolDescLower, "run code") {
			vars["HasCodeTool"] = true
			vars["CodeToolName"] = toolName
		}

		// Browser tools (browser_*, browse, navigate, etc.)
		if strings.Contains(toolNameLower, "browser") ||
			strings.Contains(toolNameLower, "browse") ||
			strings.Contains(toolNameLower, "navigate") ||
			strings.Contains(toolNameLower, "screenshot") ||
			strings.HasPrefix(toolNameLower, "browser_") ||
			strings.Contains(toolDescLower, "browse") ||
			strings.Contains(toolDescLower, "web page") {
			vars["HasBrowserTool"] = true
			vars["BrowserToolName"] = toolName
		}

		// Scrape tools (scrape, web_scrape, extract, etc.)
		if strings.Contains(toolNameLower, "scrape") ||
			strings.Contains(toolNameLower, "web_scrape") ||
			strings.Contains(toolNameLower, "extract") ||
			strings.Contains(toolDescLower, "scrape") ||
			strings.Contains(toolDescLower, "extract content") {
			vars["HasScrapeTool"] = true
			vars["ScrapeToolName"] = toolName
		}

		// Image generation tools (generate_image)
		if strings.Contains(toolNameLower, "generate_image") ||
			(strings.Contains(toolNameLower, "image") && strings.Contains(toolNameLower, "generat")) ||
			strings.Contains(toolDescLower, "generate image") ||
			strings.Contains(toolDescLower, "image generation") {
			vars["HasImageTool"] = true
			vars["HasImageGenerateTool"] = true
			vars["ImageGenerateToolName"] = toolName
		}

		// Image editing tools (edit_image)
		if strings.Contains(toolNameLower, "edit_image") ||
			(strings.Contains(toolNameLower, "image") && strings.Contains(toolNameLower, "edit")) ||
			strings.Contains(toolDescLower, "edit image") ||
			strings.Contains(toolDescLower, "image edit") {
			vars["HasImageTool"] = true
			vars["HasImageEditTool"] = true
			vars["ImageEditToolName"] = toolName
		}
	}

	vars["Tools"] = tools
	return vars
}

// CodeAssistantModule adds code-specific instructions.
type CodeAssistantModule struct {
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewCodeAssistantModule creates a new code assistant module without template service (uses fallback).
func NewCodeAssistantModule() *CodeAssistantModule {
	return &CodeAssistantModule{}
}

// NewCodeAssistantModuleWithService creates a new code assistant module with template service.
func NewCodeAssistantModuleWithService(service *prompttemplate.Service) *CodeAssistantModule {
	return &CodeAssistantModule{
		templateService: service,
	}
}

// NewCodeAssistantModuleWithModelPrompts creates a new code assistant module with model-specific template support.
func NewCodeAssistantModuleWithModelPrompts(templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *CodeAssistantModule {
	return &CodeAssistantModule{
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *CodeAssistantModule) Name() string {
	return "code_assistant"
}

// ShouldApply checks if the question is code-related.
func (m *CodeAssistantModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx != nil && promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	// Check last user message for code-related keywords.
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openai.ChatMessageRoleUser {
			content := strings.ToLower(messages[i].Content)
			if isLikelyCodeQuery(content) {
				return true
			}
			break
		}
	}
	return false
}

// Apply adds code assistant instructions.
func (m *CodeAssistantModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}

	var codeText string

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("CodeAssistantModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyCodeAssistant)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := renderTemplateContent(template.Content, map[string]any{})
			if renderErr == nil {
				codeText = rendered
				log.Debug().
					Str("module", "code_assistant").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("CodeAssistantModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "code_assistant").
					Str("rendered_content", rendered).
					Msg("CodeAssistantModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("CodeAssistantModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if codeText == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyCodeAssistant)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyCodeAssistant, map[string]any{})
			if renderErr == nil {
				codeText = rendered
				log.Debug().
					Str("module", "code_assistant").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", "global_default").
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("CodeAssistantModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "code_assistant").
					Str("rendered_content", rendered).
					Msg("CodeAssistantModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("CodeAssistantModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("CodeAssistantModule: Failed to load template from database, using fallback")
			}
		}
	}

	// Fallback to hardcoded code assistant text if template not loaded
	if codeText == "" {
		var builder strings.Builder
		builder.WriteString("When providing code assistance:\n")
		builder.WriteString("1. Provide clear, well-commented code.\n")
		builder.WriteString("2. Explain your approach and reasoning.\n")
		builder.WriteString("3. Include error handling where appropriate.\n")
		builder.WriteString("4. Follow best practices and conventions.\n")
		builder.WriteString("5. Suggest testing approaches when relevant.\n")
		builder.WriteString("6. Respect project instructions and user constraints; never violate them to simplify code.")
		codeText = builder.String()
		log.Debug().Msg("CodeAssistantModule: Using fallback hardcoded prompt")
	}

	result := appendSystemContent(messages, codeText, m.Name(), "")
	return result, nil
}

// ChainOfThoughtModule adds chain-of-thought reasoning instructions.
type ChainOfThoughtModule struct {
	templateService    *prompttemplate.Service
	modelPromptService *modelprompttemplate.Service
}

// NewChainOfThoughtModule creates a new chain-of-thought module without template service (uses fallback).
func NewChainOfThoughtModule() *ChainOfThoughtModule {
	return &ChainOfThoughtModule{}
}

// NewChainOfThoughtModuleWithService creates a new chain-of-thought module with template service.
func NewChainOfThoughtModuleWithService(service *prompttemplate.Service) *ChainOfThoughtModule {
	return &ChainOfThoughtModule{
		templateService: service,
	}
}

// NewChainOfThoughtModuleWithModelPrompts creates a new chain-of-thought module with model-specific template support.
func NewChainOfThoughtModuleWithModelPrompts(templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *ChainOfThoughtModule {
	return &ChainOfThoughtModule{
		templateService:    templateService,
		modelPromptService: modelPromptService,
	}
}

// Name returns the module identifier.
func (m *ChainOfThoughtModule) Name() string {
	return "chain_of_thought"
}

// ShouldApply checks if the question requires reasoning.
func (m *ChainOfThoughtModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx != nil && promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	// Apply for complex questions
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openai.ChatMessageRoleUser {
			content := messages[i].Content
			if isComplexQuestion(content) {
				return true
			}
			break
		}
	}
	return false
}

// Apply adds chain-of-thought instructions.
func (m *ChainOfThoughtModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}

	var cotText string

	// Try to fetch model-specific template first, then fall back to global
	if m.modelPromptService != nil && promptCtx != nil && promptCtx.ModelCatalogID != nil && *promptCtx.ModelCatalogID != "" {
		log.Debug().
			Str("model_catalog_id", *promptCtx.ModelCatalogID).
			Msg("ChainOfThoughtModule: Attempting to load model-specific template")

		template, source, err := m.modelPromptService.GetTemplateForModelByKey(ctx, *promptCtx.ModelCatalogID, prompttemplate.TemplateKeyChainOfThought)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := renderTemplateContent(template.Content, map[string]any{})
			if renderErr == nil {
				cotText = rendered
				log.Debug().
					Str("module", "chain_of_thought").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", source).
					Str("model_catalog_id", *promptCtx.ModelCatalogID).
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("ChainOfThoughtModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "chain_of_thought").
					Str("rendered_content", rendered).
					Msg("ChainOfThoughtModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("ChainOfThoughtModule: Failed to render model-specific template")
			}
		}
	}

	// Fall back to global template if model-specific not found
	if cotText == "" && m.templateService != nil {
		template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyChainOfThought)
		if err == nil && template != nil && template.IsActive {
			rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyChainOfThought, map[string]any{})
			if renderErr == nil {
				cotText = rendered
				log.Debug().
					Str("module", "chain_of_thought").
					Str("template_key", template.TemplateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", "global_default").
					Int("content_length", len(rendered)).
					Int("template_version", template.Version).
					Msg("ChainOfThoughtModule: Loaded and rendered template from database")
				log.Debug().
					Str("module", "chain_of_thought").
					Str("rendered_content", rendered).
					Msg("ChainOfThoughtModule: Rendered prompt content")
			} else {
				log.Warn().Err(renderErr).Msg("ChainOfThoughtModule: Failed to render template, using fallback")
			}
		} else {
			if err != nil {
				log.Warn().Err(err).Msg("ChainOfThoughtModule: Failed to load template from database, using fallback")
			}
		}
	}

	// Fallback to hardcoded chain-of-thought text if template not loaded
	if cotText == "" {
		var builder strings.Builder
		builder.WriteString("For complex questions, think step-by-step:\n")
		builder.WriteString("1. Break down the problem\n")
		builder.WriteString("2. Analyze each component\n")
		builder.WriteString("3. Consider different perspectives\n")
		builder.WriteString("4. Synthesize your conclusion\n")
		builder.WriteString("5. Provide a clear, structured answer")
		cotText = builder.String()
		log.Debug().Msg("ChainOfThoughtModule: Using fallback hardcoded prompt")
	}

	result := appendSystemContent(messages, cotText, m.Name(), "")
	return result, nil
}

func detectToolUsage(promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	// If the request explicitly provides tool definitions, treat tools as enabled.
	if promptCtx != nil && len(promptCtx.Tools) > 0 {
		return true
	}

	if promptCtx != nil && promptCtx.Preferences != nil {
		if useTools, ok := promptCtx.Preferences["use_tools"].(bool); ok && useTools {
			return true
		}
		// Check if image generation is requested
		if image, ok := promptCtx.Preferences["image"].(bool); ok && image {
			return true
		}
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openai.ChatMessageRoleTool {
			return true
		}
		if len(messages[i].ToolCalls) > 0 || messages[i].FunctionCall != nil {
			return true
		}
	}
	return false
}

// renderTemplateContent renders a template content string with the given variables
func renderTemplateContent(content string, variables map[string]any) (string, error) {
	if len(variables) == 0 {
		return content, nil
	}

	tmpl, err := texttemplate.New("prompt").Parse(content)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", err
	}

	return result.String(), nil
}

func isLikelyCodeQuery(content string) bool {
	if content == "" {
		return false
	}
	if strings.Contains(content, "```") {
		return true
	}
	strongSignals := []string{"func ", "function(", "class ", "package ", "import ", "console.log", "panic(", "error ", "exception", "stack trace", "traceback", "sql", "json", "yaml", "schema"}
	for _, sig := range strongSignals {
		if strings.Contains(content, sig) {
			return true
		}
	}

	if strings.Contains(content, "code of conduct") {
		return false
	}

	codeKeywords := []string{"code", "function", "implement", "debug", "bug", "syntax", "compile", "script", "api", "snippet", "library"}
	actionKeywords := []string{"write", "example", "implement", "show", "fix", "break down", "refactor", "debug", "troubleshoot"}
	keywordHit := false
	for _, keyword := range codeKeywords {
		if strings.Contains(content, keyword) {
			keywordHit = true
			break
		}
	}
	actionHit := false
	for _, act := range actionKeywords {
		if strings.Contains(content, act) {
			actionHit = true
			break
		}
	}
	return keywordHit && actionHit
}

func isComplexQuestion(content string) bool {
	if strings.TrimSpace(content) == "" {
		return false
	}
	lower := strings.ToLower(content)
	reasoningKeywords := []string{"why", "how", "explain", "analyze", "compare", "evaluate", "what if", "step by step"}
	for _, keyword := range reasoningKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	wordCount := len(strings.Fields(content))
	if wordCount >= 20 && strings.Contains(content, "?") {
		return true
	}
	if wordCount >= 30 {
		return true
	}
	return false
}
