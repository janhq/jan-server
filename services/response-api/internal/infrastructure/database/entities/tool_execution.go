package entities

import (
	"time"

	"gorm.io/datatypes"
)

// ToolExecution persists each invocation performed via MCP tools.
type ToolExecution struct {
	ID             uint           `gorm:"primaryKey"`
	ResponseID     uint           `gorm:"index"`
	CallID         string         `gorm:"size:64"`
	ToolName       string         `gorm:"size:128"`
	Arguments      datatypes.JSON `gorm:"type:jsonb"`
	Result         datatypes.JSON `gorm:"type:jsonb"`
	Status         string         `gorm:"size:32"`
	ErrorMessage   string         `gorm:"type:text"`
	ExecutionOrder int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
