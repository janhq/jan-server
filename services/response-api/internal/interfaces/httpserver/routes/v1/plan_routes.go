package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

func registerPlanRoutes(router gin.IRoutes, handler *handlers.PlanHandler) {
	// Plan routes nested under responses
	router.GET("/responses/:response_id/plan", handler.Get)
	router.GET("/responses/:response_id/plan/details", handler.GetWithDetails)
	router.GET("/responses/:response_id/plan/progress", handler.GetProgress)
	router.POST("/responses/:response_id/plan/cancel", handler.Cancel)
	router.POST("/responses/:response_id/plan/input", handler.SubmitUserInput)
	router.GET("/responses/:response_id/plan/tasks", handler.ListTasks)
}
