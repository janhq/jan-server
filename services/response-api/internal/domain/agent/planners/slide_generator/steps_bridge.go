package slide_generator

import (
	"context"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/steps"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/tool"
)

func (e *SlideGeneratorExecutor) stepsDeps() steps.ExecutorDeps {
	var callTool func(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
	if e.mcpClient != nil {
		callTool = e.mcpClient.CallTool
	}

	return steps.ExecutorDeps{
		CallTool:                                          callTool,
		ExecuteUploadSlideSpec:                            e.executeUploadSlideSpec,
		ExecuteRenderScript:                               e.executeRenderScript,
		CollectImageAssets:                                e.collectImageAssets,
		CollectDataBankText:                               e.collectDataBankText,
		CollectDataBankDatasets:                           e.collectDataBankDatasets,
		ExtractPlanAndTemplate:                            e.extractPlanAndTemplate,
		GenerateWithModel:                                 e.generateWithModel,
		GenerateWithStructuredOutput:                      e.generateWithStructuredOutput,
		GenerateWithSystemPrompt:                          e.generateWithSystemPrompt,
		GenerateWithStructuredOutputWithMaxTokens:         e.generateWithStructuredOutputWithMaxTokens,
		GenerateWithStructuredOutputWithMaxTokensAndUsage: e.generateWithStructuredOutputWithMaxTokensAndUsage,
		GenerateWithSystemPromptWithMaxTokens:             e.generateWithSystemPromptWithMaxTokens,
		EnsureSlideOrderAndID:                             ensureSlideOrderAndID,
		EnsureSlideUseComponents:                          ensureSlideUseComponents,
		TemplateLayoutIDs:                                 templateLayoutIDs,
		SlideLayoutID:                                     slideLayoutID,
		ExtractAssetIDs:                                   extractAssetIDs,
		ExtractDatasetIDs:                                 extractDatasetIDs,
		ValidateChartDatasetRefs:                          validateChartDatasetRefs,
		ValidateImageAssetRefs:                            validateImageAssetRefs,
		LayoutIDMatchesSuggestedLayout:                    layoutIDMatchesSuggestedLayout,
		SlideHasElementType:                               slideHasElementType,
		NormalizePlanIndices:                              normalizePlanIndices,
		NormalizeTemplateComponents:                       normalizeTemplateComponents,
		NormalizeTemplateLayouts:                          normalizeTemplateLayouts,
		Temperature:                                       e.temperature,
	}
}

func (e *SlideGeneratorExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecuteToolCall(ctx, e.stepsDeps(), step, input)
}

func (e *SlideGeneratorExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecuteLLMCall(ctx, e.stepsDeps(), step, input)
}

func (e *SlideGeneratorExecutor) executeReasoning(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecuteReasoning(ctx, e.stepsDeps(), params, input)
}

func (e *SlideGeneratorExecutor) executeDataBank(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecuteDataBank(ctx, e.stepsDeps(), params, input)
}

func (e *SlideGeneratorExecutor) executePlanAndTemplate(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecutePlanAndTemplate(ctx, e.stepsDeps(), params, input)
}

func (e *SlideGeneratorExecutor) executeSingleSlide(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	return steps.ExecuteSingleSlide(ctx, e.stepsDeps(), params, input)
}
