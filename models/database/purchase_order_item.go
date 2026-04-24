package database

import "time"

// PurchaseOrderItem represents a line item in a purchase order.
// Discrepancy is a GENERATED ALWAYS column: expected_qty - (received_qty + rejected_qty).
// It is read-only from the application perspective — do not set it on insert/update.
type PurchaseOrderItem struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	PurchaseOrderID string     `gorm:"column:purchase_order_id" json:"purchase_order_id"`
	ArticleSKU      string     `gorm:"column:article_sku" json:"article_sku"`
	ExpectedQty     float64    `gorm:"column:expected_qty" json:"expected_qty"`
	ReceivedQty     float64    `gorm:"column:received_qty" json:"received_qty"`
	RejectedQty     float64    `gorm:"column:rejected_qty" json:"rejected_qty"`
	UnitCost        *float64   `gorm:"column:unit_cost" json:"unit_cost,omitempty"`
	Discrepancy     *float64   `gorm:"column:discrepancy;<-:false" json:"discrepancy,omitempty"` // generated, read-only
	Notes           *string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PurchaseOrderItem) TableName() string {
	return "purchase_order_items"
}
