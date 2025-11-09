package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"jan-server/services/template-api/internal/interfaces/httpserver/handlers"
)

type sampleResponse struct {
	ID      string `json:"id" example:"sample-2"`
	Message string `json:"message" example:"Customize this implementation for real data sources"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func registerSampleRoutes(router gin.IRoutes, handler *handlers.SampleHandler) {
	router.GET("/sample", getSample(handler))
}

// getSample godoc
// @Summary      Fetch sample payload
// @Description  Demonstrates route -> handler -> domain -> repository wiring.
// @Tags         sample
// @Produce      json
// @Success      200  {object}  sampleResponse
// @Failure      500  {object}  errorResponse
// @Router       /v1/sample [get]
func getSample(handler *handlers.SampleHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := handler.GetSample(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
