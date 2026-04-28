package database

import "time"

type Lot struct {
	ID             string     `gorm:"column:id;primaryKey" json:"id"`
	LotNumber      string     `gorm:"column:lot_number" json:"lot_number"`
	SKU            string     `gorm:"column:sku" json:"sku"`
	Quantity       float64    `gorm:"column:quantity" json:"quantity"`
	ExpirationDate *time.Time `gorm:"column:expiration_date" json:"expiration_date"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Status         *string    `gorm:"column:status;default:'pending'" json:"status,omitempty"`
	// M2 extended fields
	LotNotes       *string    `gorm:"column:lot_notes" json:"lot_notes,omitempty"`
	ManufacturedAt *time.Time `gorm:"column:manufactured_at" json:"manufactured_at,omitempty"`
	BestBeforeDate *time.Time `gorm:"column:best_before_date" json:"best_before_date,omitempty"`
}

func (Lot) TableName() string {
	return "lots"
}
