package slide_creator

import (
	"jan-server/mono/apps/backend/internal/config"
	"jan-server/mono/apps/backend/internal/domain/agent/planners"
	steps "jan-server/mono/apps/backend/internal/domain/agent/planners/slide_creator/steps"
	"jan-server/mono/apps/backend/internal/domain/artifact"
	"jan-server/mono/apps/backend/internal/infrastructure/media"
)

type SlideCreatorExecutor = steps.SlideCreatorExecutor

func NewSlideCreatorExecutor(mcpClient planners.MCPClient, llmProvider planners.LLMProvider, artifactService artifact.Service, mediaClient *media.Client, cfg *config.Config) *steps.SlideCreatorExecutor {
	return steps.NewSlideCreatorExecutor(mcpClient, llmProvider, artifactService, mediaClient, cfg)
}
