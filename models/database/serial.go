package database

import "time"

type Serial struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SerialNumber string    `gorm:"column:serial_number" json:"serial_number"`
	SKU          string    `gorm:"column:sku" json:"sku"`
	Status       string    `gorm:"column:status" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Serial) TableName() string {
	return "serials"
}
