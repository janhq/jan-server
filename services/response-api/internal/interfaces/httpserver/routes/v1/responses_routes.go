package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

func registerResponseRoutes(router gin.IRoutes, handler *handlers.ResponseHandler) {
	router.POST("/responses", handler.Create)
	router.GET("/responses/:response_id", handler.Get)
	router.DELETE("/responses/:response_id", handler.Delete)
	router.POST("/responses/:response_id/cancel", handler.Cancel)
	router.GET("/responses/:response_id/input_items", handler.ListInputItems)
}
