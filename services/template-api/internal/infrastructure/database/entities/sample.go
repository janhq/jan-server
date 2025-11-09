package entities

import "time"

// Sample models the persisted representation of the sample domain entity.
type Sample struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Message   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Sample) TableName() string {
	return "samples"
}
