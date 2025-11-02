package dbschema

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel contains common fields for all database models
type BaseModel struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
