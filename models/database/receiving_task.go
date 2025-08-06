package database

import (
	"encoding/json"
	"time"
)

type ReceivingTask struct {
	ID            int             `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID        string          `gorm:"column:task_id;unique" json:"task_id"`
	InboundNumber string          `gorm:"column:inbound_number" json:"inbound_number"`
	CreatedBy     string          `gorm:"column:created_by" json:"created_by"`
	AssignedTo    *string         `gorm:"column:assigned_to" json:"assigned_to"`
	Status        string          `gorm:"column:status" json:"status"`
	Priority      string          `gorm:"column:priority" json:"priority"`
	Notes         *string         `gorm:"column:notes" json:"notes"`
	Items         json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt   *time.Time      `gorm:"column:completed_at" json:"completed_at"`
}

func (ReceivingTask) TableName() string {
	return "receiving_tasks"
}
