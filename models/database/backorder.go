package database

import "time"

// Backorder is auto-created when a picking task has insufficient stock to fulfil a SalesOrder.
// Max depth = 1: backorders cannot generate further backorders (enforced at app level).
type Backorder struct {
	ID                     string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID               string     `gorm:"column:tenant_id" json:"tenant_id"`
	OriginalSalesOrderID   string     `gorm:"column:original_sales_order_id" json:"original_sales_order_id"`
	ArticleSKU             string     `gorm:"column:article_sku" json:"article_sku"`
	RemainingQty           float64    `gorm:"column:remaining_qty" json:"remaining_qty"`
	Status                 string     `gorm:"column:status" json:"status"` // pending|fulfilled|cancelled
	GeneratedPickingTaskID *string    `gorm:"column:generated_picking_task_id" json:"generated_picking_task_id,omitempty"`
	FulfilledAt            *time.Time `gorm:"column:fulfilled_at" json:"fulfilled_at,omitempty"`
	CreatedAt              time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Backorder) TableName() string {
	return "backorders"
}
