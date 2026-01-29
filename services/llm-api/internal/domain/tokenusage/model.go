package tokenusage

import (
	"time"

	"github.com/shopspring/decimal"
)

// TokenUsage represents a single token usage record
type TokenUsage struct {
	ID               int64           `gorm:"primaryKey;autoIncrement"`
	UserID           string          `gorm:"column:user_id;not null;index"`
	ProjectID        *string         `gorm:"column:project_id;index"`
	ConversationID   *string         `gorm:"column:conversation_id"`
	Model            string          `gorm:"column:model;not null;index"`
	Provider         string          `gorm:"column:provider;not null;index"`
	PromptTokens     int             `gorm:"column:prompt_tokens;not null;default:0"`
	CompletionTokens int             `gorm:"column:completion_tokens;not null;default:0"`
	TotalTokens      int             `gorm:"column:total_tokens;not null;default:0"`
	EstimatedCostUSD decimal.Decimal `gorm:"column:estimated_cost_usd;type:decimal(10,6)"`
	RequestID        *string         `gorm:"column:request_id"`
	Stream           bool            `gorm:"column:stream;default:false"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the table name for TokenUsage
func (TokenUsage) TableName() string {
	return "llm_api.token_usage"
}

// TokenUsageDaily represents aggregated daily token usage
type TokenUsageDaily struct {
	ID                    int64           `gorm:"primaryKey;autoIncrement"`
	Date                  time.Time       `gorm:"column:date;not null;index"`
	UserID                string          `gorm:"column:user_id;not null;index"`
	ProjectID             *string         `gorm:"column:project_id"`
	Model                 string          `gorm:"column:model;not null"`
	Provider              string          `gorm:"column:provider;not null"`
	TotalPromptTokens     int64           `gorm:"column:total_prompt_tokens;not null;default:0"`
	TotalCompletionTokens int64           `gorm:"column:total_completion_tokens;not null;default:0"`
	TotalTokens           int64           `gorm:"column:total_tokens;not null;default:0"`
	RequestCount          int             `gorm:"column:request_count;not null;default:0"`
	EstimatedCostUSD      decimal.Decimal `gorm:"column:estimated_cost_usd;type:decimal(12,6)"`
	CreatedAt             time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for TokenUsageDaily
func (TokenUsageDaily) TableName() string {
	return "llm_api.token_usage_daily"
}

// UsageSummary represents aggregated usage statistics
type UsageSummary struct {
	Model                 string          `json:"model"`
	Provider              string          `json:"provider"`
	TotalPromptTokens     int64           `json:"total_prompt_tokens"`
	TotalCompletionTokens int64           `json:"total_completion_tokens"`
	TotalTokens           int64           `json:"total_tokens"`
	RequestCount          int64           `json:"request_count"`
	EstimatedCostUSD      decimal.Decimal `json:"estimated_cost_usd"`
}

// DailyAggregate represents daily aggregated usage
type DailyAggregate struct {
	Date                  time.Time       `json:"date"`
	TotalPromptTokens     int64           `json:"total_prompt_tokens"`
	TotalCompletionTokens int64           `json:"total_completion_tokens"`
	TotalTokens           int64           `json:"total_tokens"`
	RequestCount          int64           `json:"request_count"`
	EstimatedCostUSD      decimal.Decimal `json:"estimated_cost_usd"`
}

// UserUsage represents usage for a specific user
type UserUsage struct {
	UserID                string          `json:"user_id"`
	TotalPromptTokens     int64           `json:"total_prompt_tokens"`
	TotalCompletionTokens int64           `json:"total_completion_tokens"`
	TotalTokens           int64           `json:"total_tokens"`
	RequestCount          int64           `json:"request_count"`
	EstimatedCostUSD      decimal.Decimal `json:"estimated_cost_usd"`
}

// UsageFilter represents filter options for querying usage
type UsageFilter struct {
	UserID    string
	ProjectID string
	Model     string
	Provider  string
	StartDate time.Time
	EndDate   time.Time
}

// UserUsageDetail represents detailed usage for a specific user (admin view)
type UserUsageDetail struct {
	UserID                string          `json:"user_id"`
	Email                 string          `json:"email,omitempty"`
	Username              string          `json:"username,omitempty"`
	TotalPromptTokens     int64           `json:"total_prompt_tokens"`
	TotalCompletionTokens int64           `json:"total_completion_tokens"`
	TotalTokens           int64           `json:"total_tokens"`
	RequestCount          int64           `json:"request_count"`
	EstimatedCostUSD      decimal.Decimal `json:"estimated_cost_usd"`
	ByModel               []UsageSummary  `json:"by_model,omitempty" gorm:"-"`
	ByProvider            []UsageSummary  `json:"by_provider,omitempty" gorm:"-"`
}

// ActivityBucket represents a 5-minute bucket of usage activity
type ActivityBucket struct {
	BucketTime            time.Time `json:"bucket_time"`
	TotalPromptTokens     int64     `json:"total_prompt_tokens"`
	TotalCompletionTokens int64     `json:"total_completion_tokens"`
	TotalTokens           int64     `json:"total_tokens"`
	RequestCount          int64     `json:"request_count"`
}

// TokenUsage5Min represents aggregated 5-minute token usage
type TokenUsage5Min struct {
	ID                    int64     `gorm:"primaryKey;autoIncrement"`
	BucketTime            time.Time `gorm:"column:bucket_time;not null;index"`
	UserID                string    `gorm:"column:user_id;not null;index"`
	ProjectID             string    `gorm:"column:project_id;not null;default:''"`
	Model                 string    `gorm:"column:model;not null"`
	Provider              string    `gorm:"column:provider;not null"`
	TotalPromptTokens     int64     `gorm:"column:total_prompt_tokens;not null;default:0"`
	TotalCompletionTokens int64     `gorm:"column:total_completion_tokens;not null;default:0"`
	TotalTokens           int64     `gorm:"column:total_tokens;not null;default:0"`
	RequestCount          int       `gorm:"column:request_count;not null;default:0"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for TokenUsage5Min
func (TokenUsage5Min) TableName() string {
	return "llm_api.token_usage_5min"
}

// Model pricing constants (USD per token) - can be configured externally
var ModelPricing = map[string]struct {
	PromptPrice     decimal.Decimal
	CompletionPrice decimal.Decimal
}{
	"gpt-4":             {decimal.NewFromFloat(0.00003), decimal.NewFromFloat(0.00006)},
	"gpt-4-turbo":       {decimal.NewFromFloat(0.00001), decimal.NewFromFloat(0.00003)},
	"gpt-4o":            {decimal.NewFromFloat(0.000005), decimal.NewFromFloat(0.000015)},
	"gpt-4o-mini":       {decimal.NewFromFloat(0.00000015), decimal.NewFromFloat(0.0000006)},
	"gpt-3.5-turbo":     {decimal.NewFromFloat(0.0000005), decimal.NewFromFloat(0.0000015)},
	"claude-3-opus":     {decimal.NewFromFloat(0.000015), decimal.NewFromFloat(0.000075)},
	"claude-3-sonnet":   {decimal.NewFromFloat(0.000003), decimal.NewFromFloat(0.000015)},
	"claude-3-haiku":    {decimal.NewFromFloat(0.00000025), decimal.NewFromFloat(0.00000125)},
	"claude-3.5-sonnet": {decimal.NewFromFloat(0.000003), decimal.NewFromFloat(0.000015)},
}

// CalculateCost calculates estimated cost for token usage
func CalculateCost(model string, promptTokens, completionTokens int) decimal.Decimal {
	pricing, exists := ModelPricing[model]
	if !exists {
		// Default pricing for unknown models
		pricing = struct {
			PromptPrice     decimal.Decimal
			CompletionPrice decimal.Decimal
		}{
			PromptPrice:     decimal.NewFromFloat(0.000003),
			CompletionPrice: decimal.NewFromFloat(0.000006),
		}
	}

	promptCost := pricing.PromptPrice.Mul(decimal.NewFromInt(int64(promptTokens)))
	completionCost := pricing.CompletionPrice.Mul(decimal.NewFromInt(int64(completionTokens)))

	return promptCost.Add(completionCost)
}
