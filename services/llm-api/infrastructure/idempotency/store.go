package idempotency

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Record represents a cached response for an idempotent request.
type Record struct {
	Key         string    `gorm:"column:key;primaryKey"`
	PrincipalID string    `gorm:"column:principal_id"`
	Method      string    `gorm:"column:method"`
	Path        string    `gorm:"column:path"`
	Status      int       `gorm:"column:status"`
	Response    []byte    `gorm:"column:response"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

// TableName maps to the idempotency table.
func (Record) TableName() string {
	return "idempotency"
}

// Store provides persistence for idempotency records.
type Store struct {
	db *gorm.DB
}

// NewStore returns a Store instance.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Get returns a previously cached response for the given signature.
func (s *Store) Get(ctx context.Context, principalID, method, path, key string) (*Record, error) {
	var rec Record
	err := s.db.WithContext(ctx).
		Where("principal_id = ? AND method = ? AND path = ? AND key = ?", principalID, method, path, key).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "query idempotency")
	}
	return &rec, nil
}

// Save persists a new idempotent response.
func (s *Store) Save(ctx context.Context, rec *Record) error {
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		return errors.Wrap(err, "insert idempotency")
	}
	return nil
}
