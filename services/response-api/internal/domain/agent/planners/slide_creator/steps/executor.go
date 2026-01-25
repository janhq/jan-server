package steps

import (
	"context"
	"strings"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/infrastructure/media"

	"github.com/rs/zerolog/log"
)

// SlideCreatorExecutor executes steps for slide creation plans.
type SlideCreatorExecutor struct {
	mcpClient       planners.MCPClient
	llmProvider     planners.LLMProvider
	artifactService artifact.Service
	mediaClient     *media.Client
	aioBaseURL      string
	temperature     float64
}

type llmProviderWithTemperature interface {
	GenerateWithModelWithTemperature(ctx context.Context, prompt string, model string, temperature float64) (string, error)
	GenerateWithSystemPromptWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64) (string, error)
	GenerateWithStructuredOutputWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64) (string, error)
}

type llmProviderWithMaxTokens interface {
	GenerateWithModelWithMaxTokens(ctx context.Context, prompt string, model string, maxTokens *int) (string, error)
	GenerateWithSystemPromptWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, maxTokens *int) (string, error)
	GenerateWithStructuredOutputWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (string, error)
}

type llmProviderWithTemperatureAndMaxTokens interface {
	GenerateWithModelWithTemperatureAndMaxTokens(ctx context.Context, prompt string, model string, temperature float64, maxTokens *int) (string, error)
	GenerateWithSystemPromptWithTemperatureAndMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64, maxTokens *int) (string, error)
	GenerateWithStructuredOutputWithTemperatureAndMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64, maxTokens *int) (string, error)
}

// NewSlideCreatorExecutor creates a new slide creator executor.
func NewSlideCreatorExecutor(mcpClient planners.MCPClient, llmProvider planners.LLMProvider, artifactService artifact.Service, mediaClient *media.Client, cfg *config.Config) *SlideCreatorExecutor {
	aioBaseURL := ""
	if cfg != nil {
		aioBaseURL = strings.TrimSpace(cfg.AIOURL)
	}

	log.Debug().
		Str("aio_url", aioBaseURL).
		Msg("[slide_creator] executor initialized")

	return &SlideCreatorExecutor{
		mcpClient:       mcpClient,
		llmProvider:     llmProvider,
		artifactService: artifactService,
		mediaClient:     mediaClient,
		aioBaseURL:      aioBaseURL,
		temperature:     0.2,
	}
}

func (e *SlideCreatorExecutor) generateWithModel(ctx context.Context, prompt string, model string, temperature float64) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithModelWithTemperature(ctx, prompt, model, temperature)
	}
	return e.llmProvider.GenerateWithModel(ctx, prompt, model)
}

func (e *SlideCreatorExecutor) generateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithSystemPromptWithTemperature(ctx, systemPrompt, userPrompt, model, temperature)
	}
	return e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, userPrompt, model)
}

func (e *SlideCreatorExecutor) generateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithStructuredOutputWithTemperature(ctx, systemPrompt, userPrompt, model, schema, temperature)
	}
	return e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
}

func (e *SlideCreatorExecutor) generateWithStructuredOutputWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64, maxTokens *int) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperatureAndMaxTokens); ok {
		return provider.GenerateWithStructuredOutputWithTemperatureAndMaxTokens(ctx, systemPrompt, userPrompt, model, schema, temperature, maxTokens)
	}
	if provider, ok := e.llmProvider.(llmProviderWithMaxTokens); ok {
		return provider.GenerateWithStructuredOutputWithMaxTokens(ctx, systemPrompt, userPrompt, model, schema, maxTokens)
	}
	return e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
}

// CanExecute checks if this executor can handle the given action type.
func (e *SlideCreatorExecutor) CanExecute(action plan.ActionType) bool {
	switch action {
	case plan.ActionTypeToolCall, plan.ActionTypeLLMCall, plan.ActionTypeTransform, plan.ActionTypeArtifactCreate:
		return true
	default:
		return false
	}
}

// Rollback attempts to undo a step's effects.
func (e *SlideCreatorExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	return nil
}

// Execute runs a single step and returns the result.
func (e *SlideCreatorExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Str("action", string(step.Action)).Int("sequence", step.Sequence).Msg("[slide_creator] Execute started")
	switch step.Action {
	case plan.ActionTypeToolCall:
		return e.executeToolCall(ctx, step, input)
	case plan.ActionTypeLLMCall:
		return e.executeLLMCall(ctx, step, input)
	case plan.ActionTypeTransform:
		return e.executeTransform(ctx, step, input)
	case plan.ActionTypeArtifactCreate:
		return e.executeArtifactCreation(ctx, step, input)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func getModelFromContext(input agent.ExecutionInput) string {
	if input.PlanContext != nil && strings.TrimSpace(input.PlanContext.Model) != "" {
		return input.PlanContext.Model
	}
	return ""
}

var _ agent.Executor = (*SlideCreatorExecutor)(nil)
