package entities

import (
	"time"

	"gorm.io/datatypes"
)

// TableName specifies the table name for Plan.
func (Plan) TableName() string {
	return "plans"
}

// Plan represents the persisted plan record.
type Plan struct {
	ID              uint           `gorm:"primaryKey"`
	PublicID        string         `gorm:"uniqueIndex;size:64"`
	ResponseID      uint           `gorm:"index"`
	Response        *Response      `gorm:"foreignKey:ResponseID"`
	Status          string         `gorm:"size:32;index:idx_plan_status"`
	Progress        float64        `gorm:"default:0"`
	AgentType       string         `gorm:"size:32;index:idx_agent_type"`
	PlanningConfig  datatypes.JSON `gorm:"type:jsonb"`
	EstimatedSteps  int            `gorm:"default:0"`
	CompletedSteps  int            `gorm:"default:0"`
	CurrentTaskID   *uint
	FinalArtifactID *uint
	UserSelection   datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage    *string        `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time

	// Relations
	Tasks []PlanTask `gorm:"foreignKey:PlanID"`
}

// TableName specifies the table name for PlanTask.
func (PlanTask) TableName() string {
	return "plan_tasks"
}

// PlanTask represents the persisted plan task record.
type PlanTask struct {
	ID           uint    `gorm:"primaryKey"`
	PublicID     string  `gorm:"uniqueIndex;size:64"`
	PlanID       uint    `gorm:"index"`
	Plan         *Plan   `gorm:"foreignKey:PlanID"`
	Sequence     int     `gorm:"default:0"`
	TaskType     string  `gorm:"size:32"`
	Status       string  `gorm:"size:32;index:idx_task_status"`
	Title        string  `gorm:"size:256"`
	Description  *string `gorm:"type:text"`
	ErrorMessage *string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time

	// Relations
	Steps []PlanStep `gorm:"foreignKey:TaskID"`
}

// TableName specifies the table name for PlanStep.
func (PlanStep) TableName() string {
	return "plan_steps"
}

// PlanStep represents the persisted plan step record.
type PlanStep struct {
	ID            uint           `gorm:"primaryKey"`
	PublicID      string         `gorm:"uniqueIndex;size:64"`
	TaskID        uint           `gorm:"index"`
	Task          *PlanTask      `gorm:"foreignKey:TaskID"`
	Sequence      int            `gorm:"default:0"`
	Action        string         `gorm:"size:32"`
	Status        string         `gorm:"size:32;index:idx_step_status"`
	InputParams   datatypes.JSON `gorm:"type:jsonb"`
	OutputData    datatypes.JSON `gorm:"type:jsonb"`
	RetryCount    int            `gorm:"default:0"`
	MaxRetries    int            `gorm:"default:3"`
	ErrorMessage  *string        `gorm:"type:text"`
	ErrorSeverity *string        `gorm:"size:32"`
	DurationMs    *int64
	StartedAt     *time.Time
	CompletedAt   *time.Time

	// Relations
	Details []PlanStepDetail `gorm:"foreignKey:StepID"`
}

// TableName specifies the table name for PlanStepDetail.
func (PlanStepDetail) TableName() string {
	return "plan_step_details"
}

// PlanStepDetail represents the persisted plan step detail record.
type PlanStepDetail struct {
	ID                 uint      `gorm:"primaryKey"`
	PublicID           string    `gorm:"uniqueIndex;size:64"`
	StepID             uint      `gorm:"index"`
	Step               *PlanStep `gorm:"foreignKey:StepID"`
	DetailType         string    `gorm:"size:32"`
	ConversationItemID *uint
	ToolCallID         *string `gorm:"size:64"`
	ToolExecutionID    *uint
	ArtifactID         *uint
	Metadata           datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt          time.Time
}
