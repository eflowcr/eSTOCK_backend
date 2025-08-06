package database

import "time"

type Lot struct {
	ID             int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	LotNumber      string     `gorm:"column:lot_number" json:"lot_number"`
	SKU            string     `gorm:"column:sku" json:"sku"`
	Quantity       float64    `gorm:"column:quantity" json:"quantity"`
	ExpirationDate *time.Time `gorm:"column:expiration_date" json:"expiration_date"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Lot) TableName() string {
	return "lots"
}
