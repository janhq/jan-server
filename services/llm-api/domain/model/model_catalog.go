package model

import (
	"context"
	"time"

	"jan-server/services/llm-api/domain/query"

	"github.com/shopspring/decimal"
)

type SupportedParameters struct {
	Names   []string                    `json:"names"`
	Default map[string]*decimal.Decimal `json:"default"`
}

// Architecture metadata
type Architecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     *string  `json:"instruct_type"`
}

type ModelCatalogStatus string

const (
	ModelCatalogStatusInit    ModelCatalogStatus = "init"
	ModelCatalogStatusFilled  ModelCatalogStatus = "filled"
	ModelCatalogStatusUpdated ModelCatalogStatus = "updated"
)

type ModelCatalog struct {
	ID                  uint                `json:"id"`
	PublicID            string              `json:"public_id"`
	SupportedParameters SupportedParameters `json:"supported_parameters"`
	Architecture        Architecture        `json:"architecture"`
	Tags                []string            `json:"tags,omitempty"`
	Notes               *string             `json:"notes,omitempty"`
	IsModerated         *bool               `json:"is_moderated,omitempty"`
	Active              *bool               `json:"active,omitempty"`
	Extras              map[string]any      `json:"extras,omitempty"`
	Status              ModelCatalogStatus  `json:"status"`
	LastSyncedAt        *time.Time          `json:"last_synced_at,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

type ModelCatalogFilter struct {
	IDs              *[]uint
	PublicID         *string
	IsModerated      *bool
	Active           *bool
	Status           *ModelCatalogStatus
	LastSyncedAfter  *time.Time
	LastSyncedBefore *time.Time
}

type ModelCatalogRepository interface {
	Create(ctx context.Context, catalog *ModelCatalog) error
	Update(ctx context.Context, catalog *ModelCatalog) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ModelCatalog, error)
	FindByPublicID(ctx context.Context, publicID string) (*ModelCatalog, error)
	FindByFilter(ctx context.Context, filter ModelCatalogFilter, p *query.Pagination) ([]*ModelCatalog, error)
	Count(ctx context.Context, filter ModelCatalogFilter) (int64, error)
	BatchUpdateActive(ctx context.Context, filter ModelCatalogFilter, active bool) (int64, error)
	// Batch methods for optimization
	FindByIDs(ctx context.Context, ids []uint) ([]*ModelCatalog, error)
	FindByPublicIDs(ctx context.Context, publicIDs []string) ([]*ModelCatalog, error)
}
