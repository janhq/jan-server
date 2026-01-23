package slide_creator

import (
	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent/planners"
	steps "jan-server/services/response-api/internal/domain/agent/planners/slide_creator/steps"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/infrastructure/media"
)

type SlideCreatorExecutor = steps.SlideCreatorExecutor

func NewSlideCreatorExecutor(mcpClient planners.MCPClient, llmProvider planners.LLMProvider, artifactService artifact.Service, mediaClient *media.Client, cfg *config.Config) *steps.SlideCreatorExecutor {
	return steps.NewSlideCreatorExecutor(mcpClient, llmProvider, artifactService, mediaClient, cfg)
}
