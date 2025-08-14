package responses

import "time"

type InventorySerialWithSerial struct {
	ID          int
	InventoryID int
	SerialID    int
	Location    string
	CreatedAt   time.Time

	SerialID_       int `gorm:"column:serial_id"`
	SerialNumber    string
	SKU             string
	Status          string
	SerialCreatedAt time.Time
	SerialUpdatedAt time.Time
}
