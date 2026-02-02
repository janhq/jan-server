package tokenusage

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Service provides token usage business logic
type Service struct {
	repo Repository
}

// NewService creates a new token usage service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RecordUsage records a new token usage event
func (s *Service) RecordUsage(ctx context.Context, usage *TokenUsage) error {
	// Calculate cost if not provided
	if usage.EstimatedCostUSD.IsZero() {
		usage.EstimatedCostUSD = CalculateCost(usage.Model, usage.PromptTokens, usage.CompletionTokens)
	}

	// Ensure total tokens is calculated
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return s.repo.Create(ctx, usage)
}

// GetMyUsage retrieves usage summary for a user within a date range
func (s *Service) GetMyUsage(ctx context.Context, userID string, startDate, endDate time.Time) (*UsageResponse, error) {
	summaries, err := s.repo.GetUserUsage(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return s.buildUsageResponse(summaries, startDate, endDate), nil
}

// GetMyDailyUsage retrieves daily aggregated usage for a user
func (s *Service) GetMyDailyUsage(ctx context.Context, userID string, startDate, endDate time.Time) ([]DailyAggregate, error) {
	filter := UsageFilter{
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetDailyAggregates(ctx, filter)
}

// GetMyActivityUsage retrieves 5-minute bucket usage data for a user
func (s *Service) GetMyActivityUsage(ctx context.Context, userID string, startDate, endDate time.Time) ([]ActivityBucket, error) {
	return s.repo.GetActivityBuckets(ctx, userID, startDate, endDate)
}

// GetProjectUsage retrieves usage summary for a project
func (s *Service) GetProjectUsage(ctx context.Context, projectID string, startDate, endDate time.Time) (*UsageResponse, error) {
	summaries, err := s.repo.GetProjectUsage(ctx, projectID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return s.buildUsageResponse(summaries, startDate, endDate), nil
}

// GetPlatformUsage retrieves total platform usage (admin only)
func (s *Service) GetPlatformUsage(ctx context.Context, startDate, endDate time.Time) (*PlatformUsageResponse, error) {
	totalUsage, err := s.repo.GetTotalUsage(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	byModel, err := s.repo.GetUsageByModel(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	byProvider, err := s.repo.GetUsageByProvider(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	topUsers, err := s.repo.GetTopUsers(ctx, startDate, endDate, 10)
	if err != nil {
		return nil, err
	}

	return &PlatformUsageResponse{
		Period: Period{
			StartDate: startDate,
			EndDate:   endDate,
		},
		TotalUsage: *totalUsage,
		ByModel:    byModel,
		ByProvider: byProvider,
		TopUsers:   topUsers,
	}, nil
}

// GetDailyTrends retrieves daily usage trends with optional filters
func (s *Service) GetDailyTrends(ctx context.Context, filter UsageFilter) ([]DailyAggregate, error) {
	return s.repo.GetDailyAggregates(ctx, filter)
}

// GetAllUsersUsage retrieves usage for all users (admin only)
func (s *Service) GetAllUsersUsage(ctx context.Context, startDate, endDate time.Time, limit, offset int) (*AllUsersUsageResponse, error) {
	users, total, err := s.repo.GetAllUsersUsage(ctx, startDate, endDate, limit, offset)
	if err != nil {
		return nil, err
	}

	return &AllUsersUsageResponse{
		Period: Period{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Users: users,
		Total: total,
	}, nil
}

// GetAdminUserUsage retrieves detailed usage for a specific user (admin only)
func (s *Service) GetAdminUserUsage(ctx context.Context, userID string, startDate, endDate time.Time) (*UserUsageDetail, error) {
	return s.repo.GetUserUsageDetail(ctx, userID, startDate, endDate)
}

// buildUsageResponse constructs a usage response from summaries
func (s *Service) buildUsageResponse(summaries []UsageSummary, startDate, endDate time.Time) *UsageResponse {
	response := &UsageResponse{
		Period: Period{
			StartDate: startDate,
			EndDate:   endDate,
		},
		ByModel:    make([]UsageSummary, 0),
		ByProvider: make([]UsageSummary, 0),
	}

	totalPrompt := int64(0)
	totalCompletion := int64(0)
	totalTokens := int64(0)
	totalCost := decimal.Zero
	totalRequests := int64(0)

	modelMap := make(map[string]*UsageSummary)
	providerMap := make(map[string]*UsageSummary)

	for _, summary := range summaries {
		totalPrompt += summary.TotalPromptTokens
		totalCompletion += summary.TotalCompletionTokens
		totalTokens += summary.TotalTokens
		totalCost = totalCost.Add(summary.EstimatedCostUSD)
		totalRequests += summary.RequestCount

		// Aggregate by model
		if existing, ok := modelMap[summary.Model]; ok {
			existing.TotalPromptTokens += summary.TotalPromptTokens
			existing.TotalCompletionTokens += summary.TotalCompletionTokens
			existing.TotalTokens += summary.TotalTokens
			existing.EstimatedCostUSD = existing.EstimatedCostUSD.Add(summary.EstimatedCostUSD)
			existing.RequestCount += summary.RequestCount
		} else {
			modelSummary := summary
			modelSummary.Provider = ""
			modelMap[summary.Model] = &modelSummary
		}

		// Aggregate by provider
		if existing, ok := providerMap[summary.Provider]; ok {
			existing.TotalPromptTokens += summary.TotalPromptTokens
			existing.TotalCompletionTokens += summary.TotalCompletionTokens
			existing.TotalTokens += summary.TotalTokens
			existing.EstimatedCostUSD = existing.EstimatedCostUSD.Add(summary.EstimatedCostUSD)
			existing.RequestCount += summary.RequestCount
		} else {
			providerSummary := summary
			providerSummary.Model = ""
			providerMap[summary.Provider] = &providerSummary
		}
	}

	response.TotalUsage = UsageSummary{
		TotalPromptTokens:     totalPrompt,
		TotalCompletionTokens: totalCompletion,
		TotalTokens:           totalTokens,
		EstimatedCostUSD:      totalCost,
		RequestCount:          totalRequests,
	}

	for _, v := range modelMap {
		response.ByModel = append(response.ByModel, *v)
	}

	for _, v := range providerMap {
		response.ByProvider = append(response.ByProvider, *v)
	}

	return response
}

// UsageResponse represents the API response for usage queries
type UsageResponse struct {
	Period     Period         `json:"period"`
	TotalUsage UsageSummary   `json:"total_usage"`
	ByModel    []UsageSummary `json:"by_model"`
	ByProvider []UsageSummary `json:"by_provider"`
}

// PlatformUsageResponse represents admin platform-wide usage
type PlatformUsageResponse struct {
	Period     Period         `json:"period"`
	TotalUsage UsageSummary   `json:"total_usage"`
	ByModel    []UsageSummary `json:"by_model"`
	ByProvider []UsageSummary `json:"by_provider"`
	TopUsers   []UserUsage    `json:"top_users"`
}

// AllUsersUsageResponse represents paginated usage for all users (admin)
type AllUsersUsageResponse struct {
	Period Period            `json:"period"`
	Users  []UserUsageDetail `json:"users"`
	Total  int64             `json:"total"`
}

// Period represents a date range for usage queries
type Period struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
