package database

import "time"

type InventoryLot struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	InventoryID int       `gorm:"column:inventory_id" json:"inventory_id"`
	LotID       int       `gorm:"column:lot_id" json:"lot_id"`
	Quantity    float64   `gorm:"column:quantity" json:"quantity"`
	Location    string    `gorm:"column:location" json:"location"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventoryLot) TableName() string {
	return "inventory_lots"
}
