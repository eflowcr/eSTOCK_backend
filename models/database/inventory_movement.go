package database

import "time"

type InventoryMovement struct {
	ID             int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SKU            string    `gorm:"column:sku" json:"sku"`
	MovementType   string    `gorm:"column:movement_type" json:"movement_type"`
	Quantity       int       `gorm:"column:quantity" json:"quantity"`
	RemainingStock int       `gorm:"column:remaining_stock" json:"remaining_stock"`
	Reason         *string   `gorm:"column:reason" json:"reason"`
	CreatedBy      string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventoryMovement) TableName() string {
	return "inventory_movements"
}
