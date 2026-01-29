package persistence

import (
	"context"
	"time"

	"jan-server/mono/apps/backend/internal/domain/tokenusage"

	"gorm.io/gorm"
)

// TokenUsageRepository implements tokenusage.Repository using GORM
type TokenUsageRepository struct {
	db *gorm.DB
}

// NewTokenUsageRepository creates a new TokenUsageRepository
func NewTokenUsageRepository(db *gorm.DB) *TokenUsageRepository {
	return &TokenUsageRepository{db: db}
}

// Create stores a new token usage record
func (r *TokenUsageRepository) Create(ctx context.Context, usage *tokenusage.TokenUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

// GetByID retrieves a token usage record by ID
func (r *TokenUsageRepository) GetByID(ctx context.Context, id int64) (*tokenusage.TokenUsage, error) {
	var usage tokenusage.TokenUsage
	err := r.db.WithContext(ctx).First(&usage, id).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetUserUsage retrieves aggregated usage for a user within a date range
func (r *TokenUsageRepository) GetUserUsage(ctx context.Context, userID string, startDate, endDate time.Time) ([]tokenusage.UsageSummary, error) {
	var summaries []tokenusage.UsageSummary

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			model,
			provider,
			SUM(prompt_tokens) as total_prompt_tokens,
			SUM(completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startDate, endDate).
		Group("model, provider").
		Scan(&summaries).Error

	return summaries, err
}

// GetProjectUsage retrieves aggregated usage for a project within a date range
func (r *TokenUsageRepository) GetProjectUsage(ctx context.Context, projectID string, startDate, endDate time.Time) ([]tokenusage.UsageSummary, error) {
	var summaries []tokenusage.UsageSummary

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			model,
			provider,
			SUM(prompt_tokens) as total_prompt_tokens,
			SUM(completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("project_id = ? AND created_at >= ? AND created_at <= ?", projectID, startDate, endDate).
		Group("model, provider").
		Scan(&summaries).Error

	return summaries, err
}

// GetDailyAggregates retrieves daily aggregated usage based on filters
func (r *TokenUsageRepository) GetDailyAggregates(ctx context.Context, filter tokenusage.UsageFilter) ([]tokenusage.DailyAggregate, error) {
	var aggregates []tokenusage.DailyAggregate

	query := r.db.WithContext(ctx).Model(&tokenusage.TokenUsageDaily{})

	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.ProjectID != "" {
		query = query.Where("project_id = ?", filter.ProjectID)
	}
	if filter.Model != "" {
		query = query.Where("model = ?", filter.Model)
	}
	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("usage_date >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("usage_date <= ?", filter.EndDate)
	}

	err := query.
		Select(`
			usage_date as date,
			SUM(total_prompt_tokens) as total_prompt_tokens,
			SUM(total_completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(total_cost) as total_cost,
			SUM(request_count) as request_count
		`).
		Group("usage_date").
		Order("usage_date DESC").
		Scan(&aggregates).Error

	return aggregates, err
}

// GetTopUsers retrieves top users by token usage within a date range
func (r *TokenUsageRepository) GetTopUsers(ctx context.Context, startDate, endDate time.Time, limit int) ([]tokenusage.UserUsage, error) {
	var users []tokenusage.UserUsage

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			user_id,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("user_id").
		Order("total_tokens DESC").
		Limit(limit).
		Scan(&users).Error

	return users, err
}

// GetTotalUsage retrieves total platform usage within a date range
func (r *TokenUsageRepository) GetTotalUsage(ctx context.Context, startDate, endDate time.Time) (*tokenusage.UsageSummary, error) {
	var summary tokenusage.UsageSummary

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			SUM(prompt_tokens) as total_prompt_tokens,
			SUM(completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Scan(&summary).Error

	return &summary, err
}

// GetUsageByModel retrieves usage grouped by model within a date range
func (r *TokenUsageRepository) GetUsageByModel(ctx context.Context, startDate, endDate time.Time) ([]tokenusage.UsageSummary, error) {
	var summaries []tokenusage.UsageSummary

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			model,
			SUM(prompt_tokens) as total_prompt_tokens,
			SUM(completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("model").
		Order("total_tokens DESC").
		Scan(&summaries).Error

	return summaries, err
}

// GetUsageByProvider retrieves usage grouped by provider within a date range
func (r *TokenUsageRepository) GetUsageByProvider(ctx context.Context, startDate, endDate time.Time) ([]tokenusage.UsageSummary, error) {
	var summaries []tokenusage.UsageSummary

	err := r.db.WithContext(ctx).
		Model(&tokenusage.TokenUsage{}).
		Select(`
			provider,
			SUM(prompt_tokens) as total_prompt_tokens,
			SUM(completion_tokens) as total_completion_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as request_count
		`).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("provider").
		Order("total_tokens DESC").
		Scan(&summaries).Error

	return summaries, err
}
