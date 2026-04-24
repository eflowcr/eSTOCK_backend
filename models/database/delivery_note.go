package database

import "time"

// DeliveryNote represents a delivery note generated when picking completes against a SalesOrder.
// PDF generation is async (goroutine); PdfURL is set after generation.
type DeliveryNote struct {
	ID             string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID       string     `gorm:"column:tenant_id" json:"tenant_id"`
	DNNumber       string     `gorm:"column:dn_number" json:"dn_number"`
	SalesOrderID   string     `gorm:"column:sales_order_id" json:"sales_order_id"`
	PickingTaskID  *string    `gorm:"column:picking_task_id" json:"picking_task_id,omitempty"`
	CustomerID     string     `gorm:"column:customer_id" json:"customer_id"`
	TotalItems     int        `gorm:"column:total_items" json:"total_items"`
	PdfURL         *string    `gorm:"column:pdf_url" json:"pdf_url,omitempty"`
	PdfGeneratedAt *time.Time `gorm:"column:pdf_generated_at" json:"pdf_generated_at,omitempty"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at" json:"delivered_at,omitempty"`
	SignedBy       *string    `gorm:"column:signed_by" json:"signed_by,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DeliveryNote) TableName() string {
	return "delivery_notes"
}
