package steps

import (
	"context"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/tool"
)

type ExecutorDeps struct {
	CallTool                                          func(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
	ExecuteUploadSlideSpec                            func(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error)
	ExecuteRenderScript                               func(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error)
	CollectImageAssets                                func(input agent.ExecutionInput) []map[string]any
	CollectDataBankText                               func(input agent.ExecutionInput) string
	CollectDataBankDatasets                           func(input agent.ExecutionInput) []any // P1 fix: for dataset union validation
	ExtractPlanAndTemplate                            func(input agent.ExecutionInput) *schemas.PlanAndTemplate
	GenerateWithModel                                 func(ctx context.Context, prompt string, model string) (string, error)
	GenerateWithStructuredOutput                      func(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error)
	GenerateWithSystemPrompt                          func(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error)
	GenerateWithStructuredOutputWithMaxTokens         func(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (string, error)
	GenerateWithStructuredOutputWithMaxTokensAndUsage func(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (*llm.LLMResult, error)
	GenerateWithSystemPromptWithMaxTokens             func(ctx context.Context, systemPrompt string, userPrompt string, model string, maxTokens *int) (string, error)
	EnsureSlideOrderAndID                             func(slide map[string]any, slideIndex int)
	EnsureSlideUseComponents                          func(slide map[string]any)
	TemplateLayoutIDs                                 func(layouts any) map[string]bool
	SlideLayoutID                                     func(slide any) string
	ExtractAssetIDs                                   func(assets []any) map[string]bool
	ExtractDatasetIDs                                 func(datasets []any) map[string]bool
	ValidateChartDatasetRefs                          func(slide map[string]any, datasetIDs map[string]bool) string
	ValidateImageAssetRefs                            func(slide map[string]any, assetIDs map[string]bool) string
	LayoutIDMatchesSuggestedLayout                    func(layoutID string, suggestedLayout string, layouts any) bool
	SlideHasElementType                               func(slide any, elementType string) bool
	NormalizePlanIndices                              func(plan *schemas.SlidePlan)
	NormalizeTemplateComponents                       func(template *schemas.SlideTemplate)
	NormalizeTemplateLayouts                          func(plan *schemas.SlidePlan, template *schemas.SlideTemplate)
	Temperature                                       float64
}
