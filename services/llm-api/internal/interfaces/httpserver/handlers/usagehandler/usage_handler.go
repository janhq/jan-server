package usagehandler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jan-server/services/llm-api/internal/domain/tokenusage"
	"jan-server/services/llm-api/internal/domain/user"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"

	"github.com/gin-gonic/gin"
)

// UsageHandler handles token usage API requests
type UsageHandler struct {
	usageService *tokenusage.Service
	userService  *user.Service
}

// NewUsageHandler creates a new UsageHandler
func NewUsageHandler(usageService *tokenusage.Service, userService *user.Service) *UsageHandler {
	return &UsageHandler{
		usageService: usageService,
		userService:  userService,
	}
}

// GetMyUsage godoc
// @Summary Get current user's token usage
// @Description Returns token usage summary for the authenticated user within a date range
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {object} tokenusage.UsageResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/usage/me [get]
func (h *UsageHandler) GetMyUsage(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := fmt.Sprintf("%d", user.ID)

	startDate, endDate := parseDateRange(c)

	usage, err := h.usageService.GetMyUsage(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetMyDailyUsage godoc
// @Summary Get current user's daily token usage
// @Description Returns daily aggregated token usage for the authenticated user
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {array} tokenusage.DailyAggregate
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/usage/me/daily [get]
func (h *UsageHandler) GetMyDailyUsage(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := fmt.Sprintf("%d", user.ID)

	startDate, endDate := parseDateRange(c)

	dailyUsage, err := h.usageService.GetMyDailyUsage(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily usage"})
		return
	}

	c.JSON(http.StatusOK, dailyUsage)
}

// GetMyActivityUsage godoc
// @Summary Get current user's 5-minute bucket token usage
// @Description Returns 5-minute bucket aggregated token usage for the authenticated user (for charts)
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 7 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {array} tokenusage.ActivityBucket
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/usage/me/activity [get]
func (h *UsageHandler) GetMyActivityUsage(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := fmt.Sprintf("%d", user.ID)

	startDate, endDate := parseActivityDateRange(c)

	activityUsage, err := h.usageService.GetMyActivityUsage(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity usage"})
		return
	}

	c.JSON(http.StatusOK, activityUsage)
}

// GetProjectUsage godoc
// @Summary Get project's token usage
// @Description Returns token usage summary for a specific project
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {object} tokenusage.UsageResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/usage/projects/{id} [get]
func (h *UsageHandler) GetProjectUsage(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	startDate, endDate := parseDateRange(c)

	usage, err := h.usageService.GetProjectUsage(c.Request.Context(), projectID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get project usage"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetPlatformUsage godoc
// @Summary Get platform-wide token usage (Admin only)
// @Description Returns total platform token usage including top users and breakdown by model/provider
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {object} AdminPlatformUsageResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/usage [get]
func (h *UsageHandler) GetPlatformUsage(c *gin.Context) {
	startDate, endDate := parseDateRange(c)

	usage, err := h.usageService.GetPlatformUsage(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get platform usage"})
		return
	}

	response := toAdminPlatformUsageResponse(usage)
	h.enrichTopUsersWithEmail(c.Request.Context(), response)
	c.JSON(http.StatusOK, response)
}

// GetAllUsersUsage godoc
// @Summary Get usage for all users (Admin only)
// @Description Returns paginated token usage for all users within a date range
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Param limit query int false "Number of users per page, defaults to 20"
// @Param offset query int false "Offset for pagination, defaults to 0"
// @Success 200 {object} AdminAllUsersUsageResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/usage/users [get]
func (h *UsageHandler) GetAllUsersUsage(c *gin.Context) {
	startDate, endDate := parseDateRange(c)

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	usage, err := h.usageService.GetAllUsersUsage(c.Request.Context(), startDate, endDate, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users usage"})
		return
	}

	response := toAdminAllUsersUsageResponse(usage)
	h.enrichAllUsersWithEmail(c.Request.Context(), response)
	c.JSON(http.StatusOK, response)
}

// GetAdminUserUsage godoc
// @Summary Get detailed usage for a specific user (Admin only)
// @Description Returns detailed token usage for a specific user including breakdown by model/provider
// @Tags Usage
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param start_date query string false "Start date (YYYY-MM-DD), defaults to 30 days ago"
// @Param end_date query string false "End date (YYYY-MM-DD), defaults to today"
// @Success 200 {object} AdminUserUsageDetail
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/usage/users/{user_id} [get]
func (h *UsageHandler) GetAdminUserUsage(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID required"})
		return
	}

	startDate, endDate := parseDateRange(c)

	usage, err := h.usageService.GetAdminUserUsage(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user usage"})
		return
	}

	response := toAdminUserUsageDetail(usage)
	c.JSON(http.StatusOK, response)
}

// AdminUsageSummary is a cost-free usage summary for admin responses.
type AdminUsageSummary struct {
	Model                 string `json:"model,omitempty"`
	Provider              string `json:"provider,omitempty"`
	TotalPromptTokens     int64  `json:"total_prompt_tokens"`
	TotalCompletionTokens int64  `json:"total_completion_tokens"`
	TotalTokens           int64  `json:"total_tokens"`
	RequestCount          int64  `json:"request_count"`
}

// AdminUserUsage represents a top user usage entry without cost data.
type AdminUserUsage struct {
	UserID                string `json:"user_id"`
	Email                 string `json:"email,omitempty"`
	TotalPromptTokens     int64  `json:"total_prompt_tokens"`
	TotalCompletionTokens int64  `json:"total_completion_tokens"`
	TotalTokens           int64  `json:"total_tokens"`
	RequestCount          int64  `json:"request_count"`
}

// AdminUserUsageDetail represents detailed usage for a specific user without cost data.
type AdminUserUsageDetail struct {
	UserID                string              `json:"user_id"`
	Email                 string              `json:"email,omitempty"`
	Username              string              `json:"username,omitempty"`
	TotalPromptTokens     int64               `json:"total_prompt_tokens"`
	TotalCompletionTokens int64               `json:"total_completion_tokens"`
	TotalTokens           int64               `json:"total_tokens"`
	RequestCount          int64               `json:"request_count"`
	ByModel               []AdminUsageSummary `json:"by_model,omitempty"`
	ByProvider            []AdminUsageSummary `json:"by_provider,omitempty"`
}

// AdminPlatformUsageResponse represents platform usage data without estimated costs.
type AdminPlatformUsageResponse struct {
	Period     tokenusage.Period   `json:"period"`
	TotalUsage AdminUsageSummary   `json:"total_usage"`
	ByModel    []AdminUsageSummary `json:"by_model"`
	ByProvider []AdminUsageSummary `json:"by_provider"`
	TopUsers   []AdminUserUsage    `json:"top_users"`
}

// AdminAllUsersUsageResponse represents paginated user usage data without estimated costs.
type AdminAllUsersUsageResponse struct {
	Period tokenusage.Period      `json:"period"`
	Users  []AdminUserUsageDetail `json:"users"`
	Total  int64                  `json:"total"`
}

func toAdminUsageSummary(summary tokenusage.UsageSummary) AdminUsageSummary {
	return AdminUsageSummary{
		Model:                 summary.Model,
		Provider:              summary.Provider,
		TotalPromptTokens:     summary.TotalPromptTokens,
		TotalCompletionTokens: summary.TotalCompletionTokens,
		TotalTokens:           summary.TotalTokens,
		RequestCount:          summary.RequestCount,
	}
}

func toAdminUsageSummaries(summaries []tokenusage.UsageSummary) []AdminUsageSummary {
	result := make([]AdminUsageSummary, 0, len(summaries))
	for _, summary := range summaries {
		result = append(result, toAdminUsageSummary(summary))
	}
	return result
}

func toAdminUserUsage(user tokenusage.UserUsage) AdminUserUsage {
	return AdminUserUsage{
		UserID:                user.UserID,
		TotalPromptTokens:     user.TotalPromptTokens,
		TotalCompletionTokens: user.TotalCompletionTokens,
		TotalTokens:           user.TotalTokens,
		RequestCount:          user.RequestCount,
	}
}

func toAdminUserUsages(users []tokenusage.UserUsage) []AdminUserUsage {
	result := make([]AdminUserUsage, 0, len(users))
	for _, user := range users {
		result = append(result, toAdminUserUsage(user))
	}
	return result
}

func toAdminUserUsageDetailValue(detail tokenusage.UserUsageDetail) AdminUserUsageDetail {
	adminDetail := AdminUserUsageDetail{
		UserID:                detail.UserID,
		Email:                 detail.Email,
		Username:              detail.Username,
		TotalPromptTokens:     detail.TotalPromptTokens,
		TotalCompletionTokens: detail.TotalCompletionTokens,
		TotalTokens:           detail.TotalTokens,
		RequestCount:          detail.RequestCount,
	}

	if len(detail.ByModel) > 0 {
		adminDetail.ByModel = toAdminUsageSummaries(detail.ByModel)
	}

	if len(detail.ByProvider) > 0 {
		adminDetail.ByProvider = toAdminUsageSummaries(detail.ByProvider)
	}

	return adminDetail
}

func toAdminUserUsageDetail(detail *tokenusage.UserUsageDetail) *AdminUserUsageDetail {
	if detail == nil {
		return nil
	}
	adminDetail := toAdminUserUsageDetailValue(*detail)
	return &adminDetail
}

func toAdminPlatformUsageResponse(usage *tokenusage.PlatformUsageResponse) *AdminPlatformUsageResponse {
	if usage == nil {
		return nil
	}
	return &AdminPlatformUsageResponse{
		Period:     usage.Period,
		TotalUsage: toAdminUsageSummary(usage.TotalUsage),
		ByModel:    toAdminUsageSummaries(usage.ByModel),
		ByProvider: toAdminUsageSummaries(usage.ByProvider),
		TopUsers:   toAdminUserUsages(usage.TopUsers),
	}
}

func toAdminAllUsersUsageResponse(usage *tokenusage.AllUsersUsageResponse) *AdminAllUsersUsageResponse {
	if usage == nil {
		return nil
	}
	users := make([]AdminUserUsageDetail, 0, len(usage.Users))
	for _, user := range usage.Users {
		users = append(users, toAdminUserUsageDetailValue(user))
	}
	return &AdminAllUsersUsageResponse{
		Period: usage.Period,
		Users:  users,
		Total:  usage.Total,
	}
}

func (h *UsageHandler) enrichTopUsersWithEmail(ctx context.Context, response *AdminPlatformUsageResponse) {
	if response == nil || h.userService == nil || len(response.TopUsers) == 0 {
		return
	}

	for i := range response.TopUsers {
		userID := response.TopUsers[i].UserID
		parsedID, err := strconv.ParseUint(userID, 10, 64)
		if err != nil {
			continue
		}

		userProfile, err := h.userService.GetByID(ctx, uint(parsedID))
		if err != nil || userProfile == nil {
			continue
		}

		email := ""
		if userProfile.Email != nil {
			email = strings.TrimSpace(*userProfile.Email)
		}
		if email == "" && userProfile.Username != nil {
			candidate := strings.TrimSpace(*userProfile.Username)
			if strings.Contains(candidate, "@") {
				email = candidate
			}
		}
		if email == "" {
			continue
		}

		response.TopUsers[i].Email = email
	}
}

func (h *UsageHandler) enrichAllUsersWithEmail(ctx context.Context, response *AdminAllUsersUsageResponse) {
	if response == nil || h.userService == nil || len(response.Users) == 0 {
		return
	}

	for i := range response.Users {
		userID := response.Users[i].UserID
		parsedID, err := strconv.ParseUint(userID, 10, 64)
		if err != nil {
			continue
		}

		userProfile, err := h.userService.GetByID(ctx, uint(parsedID))
		if err != nil || userProfile == nil {
			continue
		}

		email := ""
		if userProfile.Email != nil {
			email = strings.TrimSpace(*userProfile.Email)
		}
		if email == "" && userProfile.Username != nil {
			candidate := strings.TrimSpace(*userProfile.Username)
			if strings.Contains(candidate, "@") {
				email = candidate
			}
		}
		if email != "" {
			response.Users[i].Email = email
		}

		// Also set username if available
		if userProfile.Username != nil {
			response.Users[i].Username = strings.TrimSpace(*userProfile.Username)
		}
	}
}

// parseDateRange extracts start and end dates from query parameters
func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -30) // Default to last 30 days

	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed.Add(24*time.Hour - time.Second) // End of day
		}
	}

	return startDate, endDate
}

// parseActivityDateRange extracts start and end dates for activity queries (default 7 days)
func parseActivityDateRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -7) // Default to last 7 days

	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed.Add(24*time.Hour - time.Second) // End of day
		}
	}

	return startDate, endDate
}
