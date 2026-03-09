package database

import "time"

type InventorySerial struct {
	ID          string    `gorm:"column:id;primaryKey" json:"id"`
	InventoryID string    `gorm:"column:inventory_id" json:"inventory_id"`
	SerialID    string    `gorm:"column:serial_id" json:"serial_id"`
	Location    string    `gorm:"column:location" json:"location"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventorySerial) TableName() string {
	return "inventory_serials"
}
