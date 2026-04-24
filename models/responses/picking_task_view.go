package responses

import (
	"encoding/json"
	"time"
)

type PickingTaskView struct {
	ID               string          `gorm:"column:id;primaryKey" json:"id"`
	TaskID           string          `gorm:"column:task_id;unique" json:"task_id"`
	OrderNumber      string          `gorm:"column:order_number" json:"order_number"`
	CreatedBy        string          `gorm:"column:created_by" json:"created_by"`
	UserCreatorName  string          `gorm:"column:user_creator_name" json:"user_creator_name"`
	AssignedTo       *string         `gorm:"column:assigned_to" json:"assigned_to"`
	UserAssigneeName *string         `gorm:"column:user_assignee_name" json:"user_assignee_name"`
	Status           string          `gorm:"column:status" json:"status"`
	Priority         string          `gorm:"column:priority" json:"priority"`
	Notes            *string         `gorm:"column:notes" json:"notes"`
	Items            json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt      *time.Time      `gorm:"column:completed_at" json:"completed_at"`
	// S2 R2 customer fields
	CustomerID   *string `gorm:"column:customer_id" json:"customer_id,omitempty"`
	CustomerCode *string `gorm:"column:customer_code" json:"customer_code,omitempty"`
	CustomerName *string `gorm:"column:customer_name" json:"customer_name,omitempty"`
	// S2.5 M3.1 tenant isolation — never leak UUID in HTTP responses
	TenantID string `gorm:"column:tenant_id" json:"-"`
}
