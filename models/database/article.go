package database

import (
	"time"
)

type Article struct {
	ID              int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SKU             string    `gorm:"column:sku;unique" json:"sku"`
	Name            string    `gorm:"column:name" json:"name"`
	Description     *string   `gorm:"column:description" json:"description"`
	UnitPrice       *float64  `gorm:"column:unit_price" json:"unit_price"`
	Presentation    string    `gorm:"column:presentation" json:"presentation"`
	TrackByLot      bool      `gorm:"column:track_by_lot" json:"track_by_lot"`
	TrackBySerial   bool      `gorm:"column:track_by_serial" json:"track_by_serial"`
	TrackExpiration bool      `gorm:"column:track_expiration" json:"track_expiration"`
	MinQuantity     *int      `gorm:"column:min_quantity" json:"min_quantity"`
	MaxQuantity     *int      `gorm:"column:max_quantity" json:"max_quantity"`
	ImageURL        *string   `gorm:"column:image_url" json:"image_url"`
	IsActive        *bool     `gorm:"column:is_active" json:"is_active"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Article) TableName() string {
	return "articles"
}
