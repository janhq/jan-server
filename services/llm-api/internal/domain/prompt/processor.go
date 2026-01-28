package prompt

import (
	"context"
	"fmt"
	"sort"

	"github.com/rs/zerolog"
	openai "github.com/sashabaranov/go-openai"

	"jan-server/services/llm-api/internal/domain/modelprompttemplate"
	"jan-server/services/llm-api/internal/domain/prompttemplate"
)

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	config  ProcessorConfig
	modules []moduleEntry
	log     zerolog.Logger
}

type moduleEntry struct {
	module   Module
	priority int
}

func modulePriority(module Module) int {
	switch module.(type) {
	case *TimingModule:
		return -15
	case *ProjectInstructionModule:
		return -10
	case *ProjectFilesModule:
		return -5 // After project instructions, before user profile
	case *UserProfileModule:
		return 5
	case *MemoryModule:
		return 10
	case *ToolInstructionsModule:
		return 20
	case *CodeAssistantModule:
		return 30
	case *ChainOfThoughtModule:
		return 40
	default:
		return 100
	case *DeepResearchModule:
		return -20 // Deep research has highest priority, runs before all other modules
	}
}

// NewProcessor creates a new prompt processor with the given configuration
// If disabled, a no-op processor is returned.
func NewProcessor(config ProcessorConfig, log zerolog.Logger) *ProcessorImpl {
	return NewProcessorWithTemplateService(config, log, nil)
}

// NewProcessorWithTemplateService creates a new prompt processor with template service support
func NewProcessorWithTemplateService(config ProcessorConfig, log zerolog.Logger, templateService *prompttemplate.Service) *ProcessorImpl {
	return NewProcessorWithServices(config, log, templateService, nil)
}

// NewProcessorWithServices creates a new prompt processor with both template and model-specific template services
func NewProcessorWithServices(config ProcessorConfig, log zerolog.Logger, templateService *prompttemplate.Service, modelPromptService *modelprompttemplate.Service) *ProcessorImpl {
	processor := &ProcessorImpl{
		config:  config,
		modules: make([]moduleEntry, 0),
		log:     log.With().Str("component", "prompt-processor").Logger(),
	}

	if !config.Enabled {
		processor.log = processor.log.With().Str("mode", "noop").Logger()
		return processor
	}

	// Always register timing module for AI assistant intro and current date
	// Use model-specific template service if available
	if templateService != nil && modelPromptService != nil {
		processor.log.Debug().Msg("registering TimingModule with model-specific template support")
		processor.RegisterModule(NewTimingModuleWithModelPrompts(templateService, modelPromptService))
	} else if templateService != nil {
		processor.log.Debug().Msg("registering TimingModule with template service")
		processor.RegisterModule(NewTimingModuleWithService(templateService))
	} else {
		processor.RegisterModule(NewTimingModule())
	}

	processor.RegisterModule(NewProjectInstructionModule())

	// Register UserProfileModule with model-specific template service if available
	if templateService != nil && modelPromptService != nil {
		processor.log.Debug().Msg("registering UserProfileModule with model-specific template support")
		processor.RegisterModule(NewUserProfileModuleWithModelPrompts(templateService, modelPromptService))
	} else if templateService != nil {
		processor.log.Debug().Msg("registering UserProfileModule with template service")
		processor.RegisterModule(NewUserProfileModuleWithService(templateService))
	} else {
		processor.RegisterModule(NewUserProfileModule())
	}

	// Register modules based on configuration
	if config.EnableMemory {
		if templateService != nil && modelPromptService != nil {
			processor.log.Debug().Msg("registering MemoryModule with model-specific template support")
			processor.RegisterModule(NewMemoryModuleWithModelPrompts(true, templateService, modelPromptService))
		} else if templateService != nil {
			processor.log.Debug().Msg("registering MemoryModule with template service")
			processor.RegisterModule(NewMemoryModuleWithService(true, templateService))
		} else {
			processor.RegisterModule(NewMemoryModule(true))
		}
	}

	if config.EnableTools {
		if templateService != nil && modelPromptService != nil {
			processor.log.Debug().Msg("registering ToolInstructionsModule with model-specific template support")
			processor.RegisterModule(NewToolInstructionsModuleWithModelPrompts(true, templateService, modelPromptService))
		} else if templateService != nil {
			processor.log.Debug().Msg("registering ToolInstructionsModule with template service")
			processor.RegisterModule(NewToolInstructionsModuleWithService(true, templateService))
		} else {
			processor.RegisterModule(NewToolInstructionsModule(true))
		}
	}

	// Conditional template-based modules (CoT, code assistant)
	if config.EnableTemplates {
		if templateService != nil && modelPromptService != nil {
			processor.log.Debug().Msg("registering CodeAssistantModule with model-specific template support")
			processor.RegisterModule(NewCodeAssistantModuleWithModelPrompts(templateService, modelPromptService))
			processor.log.Debug().Msg("registering ChainOfThoughtModule with model-specific template support")
			processor.RegisterModule(NewChainOfThoughtModuleWithModelPrompts(templateService, modelPromptService))
		} else if templateService != nil {
			processor.log.Debug().Msg("registering CodeAssistantModule with template service")
			processor.RegisterModule(NewCodeAssistantModuleWithService(templateService))
			processor.log.Debug().Msg("registering ChainOfThoughtModule with template service")
			processor.RegisterModule(NewChainOfThoughtModuleWithService(templateService))
		} else {
			processor.RegisterModule(NewCodeAssistantModule())
			processor.RegisterModule(NewChainOfThoughtModule())
		}
	}

	return processor
}

// RegisterModule adds a module to the processor
func (p *ProcessorImpl) RegisterModule(module Module) {
	entry := moduleEntry{
		module:   module,
		priority: modulePriority(module),
	}
	p.modules = append(p.modules, entry)
	sort.Slice(p.modules, func(i, j int) bool {
		return p.modules[i].priority < p.modules[j].priority
	})
	p.log.Debug().Str("module", module.Name()).Int("priority", entry.priority).Msg("registered prompt module")
}

// Process applies all relevant modules to the messages
func (p *ProcessorImpl) Process(
	ctx context.Context,
	promptCtx *Context,
	messages []openai.ChatCompletionMessage,
) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil && ctx.Err() != nil {
		return messages, ctx.Err()
	}
	if promptCtx == nil {
		promptCtx = &Context{}
	}
	if !p.config.Enabled {
		return messages, nil
	}
	if len(messages) == 0 {
		return messages, nil
	}

	result := messages
	appliedModules := make([]string, 0, len(p.modules))

	for idx, entry := range p.modules {
		if ctx != nil && ctx.Err() != nil {
			p.log.Warn().Err(ctx.Err()).Msg("context cancelled during prompt processing")
			return result, ctx.Err()
		}

		if isModuleDisabled(promptCtx.Preferences, entry.module.Name()) {
			p.log.Debug().
				Str("module", entry.module.Name()).
				Str("conversation_id", promptCtx.ConversationID).
				Msg("prompt module disabled via preferences")
			continue
		}

		if entry.module.ShouldApply(ctx, promptCtx, result) {
			before := result
			var err error
			result, err = entry.module.Apply(ctx, promptCtx, result)
			if err != nil {
				p.log.Error().
					Err(err).
					Str("module", entry.module.Name()).
					Str("position", fmt.Sprintf("%d/%d", idx+1, len(p.modules))).
					Msg("failed to apply prompt module")
				return before, err
			}
			if result == nil {
				return before, fmt.Errorf("module %s returned nil messages", entry.module.Name())
			}
			appliedModules = append(appliedModules, entry.module.Name())
		}
	}

	if len(appliedModules) > 0 {
		promptCtx.AppliedModules = append([]string(nil), appliedModules...)
		p.log.Debug().
			Strs("applied_modules", appliedModules).
			Str("conversation_id", promptCtx.ConversationID).
			Msg("applied prompt orchestration modules")
	} else {
		promptCtx.AppliedModules = nil
	}

	return result, nil
}
