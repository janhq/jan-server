package tokenusage

import (
	"context"
	"time"
)

// Repository defines the interface for token usage data access
type Repository interface {
	// Create stores a new token usage record
	Create(ctx context.Context, usage *TokenUsage) error

	// GetByID retrieves a token usage record by ID
	GetByID(ctx context.Context, id int64) (*TokenUsage, error)

	// GetUserUsage retrieves aggregated usage for a user within a date range
	GetUserUsage(ctx context.Context, userID string, startDate, endDate time.Time) ([]UsageSummary, error)

	// GetProjectUsage retrieves aggregated usage for a project within a date range
	GetProjectUsage(ctx context.Context, projectID string, startDate, endDate time.Time) ([]UsageSummary, error)

	// GetDailyAggregates retrieves daily aggregated usage based on filters
	GetDailyAggregates(ctx context.Context, filter UsageFilter) ([]DailyAggregate, error)

	// GetTopUsers retrieves top users by token usage within a date range
	GetTopUsers(ctx context.Context, startDate, endDate time.Time, limit int) ([]UserUsage, error)

	// GetTotalUsage retrieves total platform usage within a date range
	GetTotalUsage(ctx context.Context, startDate, endDate time.Time) (*UsageSummary, error)

	// GetUsageByModel retrieves usage grouped by model within a date range
	GetUsageByModel(ctx context.Context, startDate, endDate time.Time) ([]UsageSummary, error)

	// GetUsageByProvider retrieves usage grouped by provider within a date range
	GetUsageByProvider(ctx context.Context, startDate, endDate time.Time) ([]UsageSummary, error)

	// GetActivityBuckets retrieves 5-minute bucket usage data for a user
	GetActivityBuckets(ctx context.Context, userID string, startDate, endDate time.Time) ([]ActivityBucket, error)

	// GetAllUsersUsage retrieves aggregated usage for all users within a date range (admin)
	GetAllUsersUsage(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]UserUsageDetail, int64, error)

	// GetUserUsageDetail retrieves detailed usage for a specific user (admin)
	GetUserUsageDetail(ctx context.Context, userID string, startDate, endDate time.Time) (*UserUsageDetail, error)
}
