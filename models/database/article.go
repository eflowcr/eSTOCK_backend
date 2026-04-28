package database

import (
	"time"
)

type Article struct {
	ID               string    `gorm:"column:id;primaryKey" json:"id"`
	SKU              string    `gorm:"column:sku;unique" json:"sku"`
	Name             string    `gorm:"column:name" json:"name"`
	Description      *string   `gorm:"column:description" json:"description"`
	UnitPrice        *float64  `gorm:"column:unit_price" json:"unit_price"`
	Presentation     string    `gorm:"column:presentation" json:"presentation"`
	TrackByLot       bool      `gorm:"column:track_by_lot" json:"track_by_lot"`
	TrackBySerial    bool      `gorm:"column:track_by_serial" json:"track_by_serial"`
	TrackExpiration  bool      `gorm:"column:track_expiration" json:"track_expiration"`
	RotationStrategy string    `gorm:"column:rotation_strategy" json:"rotation_strategy"`
	MinQuantity      *int      `gorm:"column:min_quantity" json:"min_quantity"`
	MaxQuantity      *int      `gorm:"column:max_quantity" json:"max_quantity"`
	ImageURL         *string   `gorm:"column:image_url" json:"image_url"`
	IsActive         *bool     `gorm:"column:is_active" json:"is_active"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	// M2 extended fields
	CategoryID          *string  `gorm:"column:category_id" json:"category_id,omitempty"`
	ShelfLifeInDays     *int     `gorm:"column:shelf_life_in_days" json:"shelf_life_in_days,omitempty"`
	SafetyStock         float64  `gorm:"column:safety_stock;default:0" json:"safety_stock"`
	BatchNumberSeries   *string  `gorm:"column:batch_number_series" json:"batch_number_series,omitempty"`
	SerialNumberSeries  *string  `gorm:"column:serial_number_series" json:"serial_number_series,omitempty"`
	MinOrderQty         float64  `gorm:"column:min_order_qty;default:0" json:"min_order_qty"`
	DefaultLocationID   *string  `gorm:"column:default_location_id" json:"default_location_id,omitempty"`
	ReceivingNotes      *string  `gorm:"column:receiving_notes" json:"receiving_notes,omitempty"`
	ShippingNotes       *string  `gorm:"column:shipping_notes" json:"shipping_notes,omitempty"`
}

func (Article) TableName() string {
	return "articles"
}
