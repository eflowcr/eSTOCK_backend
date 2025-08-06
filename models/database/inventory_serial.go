package database

import "time"

type InventorySerial struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	InventoryID int       `gorm:"column:inventory_id" json:"inventory_id"`
	SerialID    int       `gorm:"column:serial_id" json:"serial_id"`
	Location    string    `gorm:"column:location" json:"location"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventorySerial) TableName() string {
	return "inventory_serials"
}
