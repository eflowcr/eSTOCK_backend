package database

import "time"

type InventoryMovement struct {
	ID             string    `gorm:"column:id;primaryKey" json:"id"`
	SKU            string    `gorm:"column:sku" json:"sku"`
	Location       string    `gorm:"column:location" json:"location"`
	MovementType   string    `gorm:"column:movement_type" json:"movement_type"`
	Quantity       float64   `gorm:"column:quantity" json:"quantity"`
	RemainingStock float64   `gorm:"column:remaining_stock" json:"remaining_stock"`
	Reason         *string   `gorm:"column:reason" json:"reason"`
	CreatedBy      string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// S2 trazabilidad extendida (migration 000018)
	ReferenceType *string  `gorm:"column:reference_type" json:"reference_type,omitempty"`
	ReferenceID   *string  `gorm:"column:reference_id" json:"reference_id,omitempty"`
	LotID         *string  `gorm:"column:lot_id" json:"lot_id,omitempty"`
	SerialID      *string  `gorm:"column:serial_id" json:"serial_id,omitempty"`
	UnitCost      *float64 `gorm:"column:unit_cost" json:"unit_cost,omitempty"`
	BeforeQty     *float64 `gorm:"column:before_qty" json:"before_qty,omitempty"`
	AfterQty      *float64 `gorm:"column:after_qty" json:"after_qty,omitempty"`
	UserID        *string  `gorm:"column:user_id" json:"user_id,omitempty"`
}

func (InventoryMovement) TableName() string {
	return "inventory_movements"
}
