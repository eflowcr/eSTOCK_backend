package responses

import "time"

type InventoryMovementView struct {
	SKU            string    `gorm:"column:sku" json:"sku"`
	Description    string    `gorm:"column:description" json:"description"`
	Location       string    `gorm:"column:location" json:"location"`
	MovementType   string    `gorm:"column:movement_type" json:"movement_type"`
	Quantity       float64   `gorm:"column:quantity" json:"quantity"`
	RemainingStock float64   `gorm:"column:remaining_stock" json:"remaining_stock"`
	Reason         *string   `gorm:"column:reason" json:"reason"`
	CreatedBy      string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
