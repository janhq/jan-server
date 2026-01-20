package slide_generator

import (
	"context"
	"strings"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/infrastructure/media"

	"github.com/rs/zerolog/log"
)

// SlideGeneratorExecutor executes steps for slide generation plans.
type SlideGeneratorExecutor struct {
	mcpClient          planners.MCPClient
	llmProvider        planners.LLMProvider
	artifactService    artifact.Service
	mediaClient        *media.Client
	skillExecutor      *planners.SkillExecutor
	aioClient          *agent.AIOSandboxClient // Direct AIO sandbox client (bypasses MCP)
	aioBaseURL         string
	rendererScriptPath string
	rendererEnabled    bool
	temperature        float64 // LLM temperature for slide generation (default: 0.2)
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

// llmProviderWithUsage supports returning token usage from structured output calls.
type llmProviderWithUsage interface {
	GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (*llm.LLMResult, error)
	GenerateWithStructuredOutputWithTemperatureAndMaxTokensAndUsage(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64, maxTokens *int) (*llm.LLMResult, error)
}

// NewSlideGeneratorExecutor creates a new slide generator executor.
func NewSlideGeneratorExecutor(mcpClient planners.MCPClient, llmProvider planners.LLMProvider, artifactService artifact.Service, mediaClient *media.Client, skillExecutor *planners.SkillExecutor, cfg *config.Config) *SlideGeneratorExecutor {
	aioBaseURL := ""
	rendererScriptPath := ""
	rendererEnabled := true
	if cfg != nil {
		aioBaseURL = strings.TrimSpace(cfg.AIOURL)
		rendererScriptPath = strings.TrimSpace(cfg.SlideRendererScript)
		rendererEnabled = cfg.SlideRendererEnabled
	}

	// Initialize direct AIO sandbox client (bypasses unstable MCP layer)
	var aioClient *agent.AIOSandboxClient
	if aioBaseURL != "" {
		aioClient = agent.NewAIOSandboxClient(aioBaseURL, log.Logger)
		log.Info().Str("aio_url", aioBaseURL).Msg("[slide_generator] Initialized direct AIO sandbox client")
	}

	log.Debug().
		Bool("renderer_enabled", rendererEnabled).
		Str("renderer_script_path", rendererScriptPath).
		Bool("aio_configured", aioBaseURL != "").
		Msg("[slide_generator] executor initialized")

	return &SlideGeneratorExecutor{
		mcpClient:          mcpClient,
		llmProvider:        llmProvider,
		artifactService:    artifactService,
		mediaClient:        mediaClient,
		skillExecutor:      skillExecutor,
		aioClient:          aioClient,
		aioBaseURL:         aioBaseURL,
		rendererScriptPath: rendererScriptPath,
		rendererEnabled:    rendererEnabled,
		temperature:        0.2, // Low temperature for deterministic, structured output
	}
}

func (e *SlideGeneratorExecutor) generateWithModel(ctx context.Context, prompt string, model string) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithModelWithTemperature(ctx, prompt, model, e.temperature)
	}
	return e.llmProvider.GenerateWithModel(ctx, prompt, model)
}

func (e *SlideGeneratorExecutor) generateWithModelWithMaxTokens(ctx context.Context, prompt string, model string, maxTokens *int) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperatureAndMaxTokens); ok {
		return provider.GenerateWithModelWithTemperatureAndMaxTokens(ctx, prompt, model, e.temperature, maxTokens)
	}
	if provider, ok := e.llmProvider.(llmProviderWithMaxTokens); ok {
		return provider.GenerateWithModelWithMaxTokens(ctx, prompt, model, maxTokens)
	}
	return e.llmProvider.GenerateWithModel(ctx, prompt, model)
}

func (e *SlideGeneratorExecutor) generateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithSystemPromptWithTemperature(ctx, systemPrompt, userPrompt, model, e.temperature)
	}
	return e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, userPrompt, model)
}

func (e *SlideGeneratorExecutor) generateWithSystemPromptWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, maxTokens *int) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperatureAndMaxTokens); ok {
		return provider.GenerateWithSystemPromptWithTemperatureAndMaxTokens(ctx, systemPrompt, userPrompt, model, e.temperature, maxTokens)
	}
	if provider, ok := e.llmProvider.(llmProviderWithMaxTokens); ok {
		return provider.GenerateWithSystemPromptWithMaxTokens(ctx, systemPrompt, userPrompt, model, maxTokens)
	}
	return e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, userPrompt, model)
}

func (e *SlideGeneratorExecutor) generateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithStructuredOutputWithTemperature(ctx, systemPrompt, userPrompt, model, schema, e.temperature)
	}
	return e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
}

func (e *SlideGeneratorExecutor) generateWithStructuredOutputWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperatureAndMaxTokens); ok {
		return provider.GenerateWithStructuredOutputWithTemperatureAndMaxTokens(ctx, systemPrompt, userPrompt, model, schema, e.temperature, maxTokens)
	}
	if provider, ok := e.llmProvider.(llmProviderWithMaxTokens); ok {
		return provider.GenerateWithStructuredOutputWithMaxTokens(ctx, systemPrompt, userPrompt, model, schema, maxTokens)
	}
	return e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
}

// generateWithStructuredOutputWithMaxTokensAndUsage generates structured output and returns token usage.
func (e *SlideGeneratorExecutor) generateWithStructuredOutputWithMaxTokensAndUsage(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (*llm.LLMResult, error) {
	// Try the usage-returning interface first
	if provider, ok := e.llmProvider.(llmProviderWithUsage); ok {
		return provider.GenerateWithStructuredOutputWithTemperatureAndMaxTokensAndUsage(ctx, systemPrompt, userPrompt, model, schema, e.temperature, maxTokens)
	}
	// Fallback to non-usage returning call
	content, err := e.generateWithStructuredOutputWithMaxTokens(ctx, systemPrompt, userPrompt, model, schema, maxTokens)
	if err != nil {
		return nil, err
	}
	return &llm.LLMResult{Content: content, Usage: nil}, nil
}

// CanExecute checks if this executor can handle the given action type.
func (e *SlideGeneratorExecutor) CanExecute(action plan.ActionType) bool {
	switch action {
	case plan.ActionTypeToolCall, plan.ActionTypeLLMCall, plan.ActionTypeSkillExecute, plan.ActionTypeArtifactCreate:
		return true
	default:
		return false
	}
}

// Rollback attempts to undo a step's effects.
func (e *SlideGeneratorExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	if step.Action == plan.ActionTypeArtifactCreate {
		return nil
	}
	return nil
}

// Execute runs a single step and returns the result.
func (e *SlideGeneratorExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Str("action", string(step.Action)).Int("sequence", step.Sequence).Msg("[slide_generator] Execute started")
	switch step.Action {
	case plan.ActionTypeToolCall:
		return e.executeToolCall(ctx, step, input)
	case plan.ActionTypeLLMCall:
		return e.executeLLMCall(ctx, step, input)
	case plan.ActionTypeSkillExecute:
		if e.skillExecutor == nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "SKILL_EXECUTOR_MISSING",
					Message:  "skill executor not configured",
					Severity: status.ErrorSeverityFatal,
				},
			}, nil
		}
		return e.skillExecutor.Execute(ctx, step, input)
	case plan.ActionTypeArtifactCreate:
		return e.executeArtifactCreation(ctx, step, input)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}
