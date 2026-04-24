package database

import "time"

// SalesOrderItem represents a line item in a sales order.
// PickedQty is updated as the associated PickingTask progresses.
type SalesOrderItem struct {
	ID            string    `gorm:"column:id;primaryKey" json:"id"`
	SalesOrderID  string    `gorm:"column:sales_order_id" json:"sales_order_id"`
	ArticleSKU    string    `gorm:"column:article_sku" json:"article_sku"`
	ExpectedQty   float64   `gorm:"column:expected_qty" json:"expected_qty"`
	PickedQty     float64   `gorm:"column:picked_qty" json:"picked_qty"`
	UnitPrice     *float64  `gorm:"column:unit_price" json:"unit_price,omitempty"`
	Notes         *string   `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SalesOrderItem) TableName() string {
	return "sales_order_items"
}
