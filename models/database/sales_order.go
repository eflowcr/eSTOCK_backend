package database

import "time"

// SalesOrder represents a sales order header (draft→submitted→partial→completed|cancelled).
// When submitted, a PickingTask is auto-generated and linked via PickingTaskID.
type SalesOrder struct {
	ID            string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID      string     `gorm:"column:tenant_id" json:"tenant_id"`
	SONumber      string     `gorm:"column:so_number" json:"so_number"`
	CustomerID    string     `gorm:"column:customer_id" json:"customer_id"`
	Status        string     `gorm:"column:status" json:"status"`
	ExpectedDate  *time.Time `gorm:"column:expected_date" json:"expected_date,omitempty"`
	Notes         *string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedBy     *string    `gorm:"column:created_by" json:"created_by,omitempty"`
	SubmittedAt   *time.Time `gorm:"column:submitted_at" json:"submitted_at,omitempty"`
	CompletedAt   *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CancelledAt   *time.Time `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`
	PickingTaskID *string    `gorm:"column:picking_task_id" json:"picking_task_id,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (SalesOrder) TableName() string {
	return "sales_orders"
}
