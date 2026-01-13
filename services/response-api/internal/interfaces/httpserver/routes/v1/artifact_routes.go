package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

func registerArtifactRoutes(router gin.IRoutes, handler *handlers.ArtifactHandler) {
	// Artifact routes
	router.GET("/artifacts/:artifact_id", handler.Get)
	router.GET("/artifacts/:artifact_id/versions", handler.GetVersions)
	router.GET("/artifacts/:artifact_id/download", handler.Download)
	router.DELETE("/artifacts/:artifact_id", handler.Delete)

	// Artifact routes nested under responses
	router.GET("/responses/:response_id/artifacts", handler.GetByResponse)
	router.GET("/responses/:response_id/artifacts/latest", handler.GetLatestByResponse)
}
