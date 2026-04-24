package database

import "time"

type Adjustment struct {
	ID               string    `gorm:"column:id;primaryKey" json:"id"`
	SKU              string    `gorm:"column:sku" json:"sku"`
	Location         string    `gorm:"column:location" json:"location"`
	PreviousQuantity int       `gorm:"column:previous_quantity" json:"previous_quantity"`
	AdjustmentQty    int       `gorm:"column:adjustment_quantity" json:"adjustment_quantity"`
	NewQuantity      int       `gorm:"column:new_quantity" json:"new_quantity"`
	Reason           string    `gorm:"column:reason" json:"reason"`
	Notes            *string   `gorm:"column:notes" json:"notes"`
	UserID           string    `gorm:"column:user_id" json:"user_id"`
	AdjustmentType   string    `gorm:"column:adjustment_type" json:"adjustment_type"` // S2 D1: increase|decrease|count_reconcile
	TenantID         string    `gorm:"column:tenant_id" json:"tenant_id"`             // S2.5 M3.1: tenant isolation
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Adjustment) TableName() string {
	return "adjustments"
}
