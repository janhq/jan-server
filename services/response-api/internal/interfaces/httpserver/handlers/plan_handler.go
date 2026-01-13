package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/interfaces/httpserver/requests"
	"jan-server/services/response-api/internal/interfaces/httpserver/responses"
)

// PlanHandler exposes HTTP entrypoints for the Plans API.
type PlanHandler struct {
	service plan.Service
	log     zerolog.Logger
}

// NewPlanHandler constructs the handler.
func NewPlanHandler(service plan.Service, log zerolog.Logger) *PlanHandler {
	return &PlanHandler{
		service: service,
		log:     log.With().Str("handler", "plan").Logger(),
	}
}

// Get handles GET /v1/responses/:response_id/plan
// @Summary Get plan for a response
// @Description Retrieves the execution plan associated with a response
// @Tags Plans
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.PlanResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan [get]
func (h *PlanHandler) Get(c *gin.Context) {
	responseID := c.Param("response_id")

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	c.JSON(http.StatusOK, responses.MapPlanToResponse(p))
}

// GetWithDetails handles GET /v1/responses/:response_id/plan/details
// @Summary Get plan with full details
// @Description Retrieves the execution plan with all tasks and steps
// @Tags Plans
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.PlanDetailResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan/details [get]
func (h *PlanHandler) GetWithDetails(c *gin.Context) {
	responseID := c.Param("response_id")

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	fullPlan, err := h.service.GetPlanWithDetails(c.Request.Context(), p.ID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan details")
		return
	}

	c.JSON(http.StatusOK, responses.MapPlanDetailToResponse(fullPlan))
}

// GetProgress handles GET /v1/responses/:response_id/plan/progress
// @Summary Get plan progress
// @Description Retrieves the current progress of a plan execution
// @Tags Plans
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.PlanProgressResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan/progress [get]
func (h *PlanHandler) GetProgress(c *gin.Context) {
	responseID := c.Param("response_id")

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	progress, err := h.service.GetProgress(c.Request.Context(), p.ID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan progress")
		return
	}

	c.JSON(http.StatusOK, responses.MapPlanProgressToResponse(progress))
}

// Cancel handles POST /v1/responses/:response_id/plan/cancel
// @Summary Cancel plan execution
// @Description Cancels an in-progress plan execution
// @Tags Plans
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Param request body requests.CancelPlanRequest false "Cancel request"
// @Success 200 {object} responses.PlanResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan/cancel [post]
func (h *PlanHandler) Cancel(c *gin.Context) {
	responseID := c.Param("response_id")

	var req requests.CancelPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req.Reason = "Cancelled by user"
	}

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	if err := h.service.Cancel(c.Request.Context(), p.ID, req.Reason); err != nil {
		responses.HandleError(c, err, "failed to cancel plan")
		return
	}

	// Fetch updated plan
	p, err = h.service.GetByID(c.Request.Context(), p.ID)
	if err != nil {
		responses.HandleError(c, err, "failed to get updated plan")
		return
	}

	c.JSON(http.StatusOK, responses.MapPlanToResponse(p))
}

// SubmitUserInput handles POST /v1/responses/:response_id/plan/input
// @Summary Submit user input to resume plan
// @Description Submits user selection or input to resume a waiting plan
// @Tags Plans
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Param request body requests.UserInputRequest true "User input"
// @Success 200 {object} responses.PlanResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan/input [post]
func (h *PlanHandler) SubmitUserInput(c *gin.Context) {
	responseID := c.Param("response_id")

	var req requests.UserInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	if p.Status != status.StatusWaitForUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan is not waiting for user input"})
		return
	}

	if req.Selection != "" {
		if err := h.service.SetUserSelection(c.Request.Context(), p.ID, req.Selection); err != nil {
			responses.HandleError(c, err, "failed to set user selection")
			return
		}
	}

	// Resume plan
	if err := h.service.UpdateStatus(c.Request.Context(), p.ID, status.StatusInProgress, nil); err != nil {
		responses.HandleError(c, err, "failed to resume plan")
		return
	}

	// Fetch updated plan
	p, err = h.service.GetByID(c.Request.Context(), p.ID)
	if err != nil {
		responses.HandleError(c, err, "failed to get updated plan")
		return
	}

	c.JSON(http.StatusOK, responses.MapPlanToResponse(p))
}

// ListTasks handles GET /v1/responses/:response_id/plan/tasks
// @Summary List plan tasks
// @Description Retrieves all tasks for a plan
// @Tags Plans
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {array} responses.TaskResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/plan/tasks [get]
func (h *PlanHandler) ListTasks(c *gin.Context) {
	responseID := c.Param("response_id")

	p, err := h.service.GetByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan")
		return
	}

	fullPlan, err := h.service.GetPlanWithDetails(c.Request.Context(), p.ID)
	if err != nil {
		responses.HandleError(c, err, "failed to get plan tasks")
		return
	}

	taskResponses := make([]responses.TaskResponse, 0, len(fullPlan.Tasks))
	for _, task := range fullPlan.Tasks {
		taskResponses = append(taskResponses, responses.MapTaskToResponse(&task))
	}

	c.JSON(http.StatusOK, taskResponses)
}
