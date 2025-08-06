package database

import (
	"time"
)

type Inventory struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SKU          string    `gorm:"column:sku;index:sku_location_idx" json:"sku"`
	Name         string    `gorm:"column:name" json:"name"`
	Description  *string   `gorm:"column:description" json:"description"`
	Location     string    `gorm:"column:location;index:sku_location_idx" json:"location"`
	Quantity     float64   `gorm:"column:quantity" json:"quantity"`
	Status       string    `gorm:"column:status" json:"status"`
	Presentation string    `gorm:"column:presentation" json:"presentation"`
	UnitPrice    *float64  `gorm:"column:unit_price" json:"unit_price"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Inventory) TableName() string {
	return "inventory"
}
