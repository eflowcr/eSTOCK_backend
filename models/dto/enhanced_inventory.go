package dto

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
)

type EnhancedInventory struct {
	ID              int               `json:"id"`
	SKU             string            `json:"sku"`
	Location        string            `json:"location"`
	Quantity        float64           `json:"quantity"`
	Status          string            `json:"status"`
	UnitPrice       float64           `json:"unit_price"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Presentation    string            `json:"presentation"`
	TrackByLot      bool              `json:"track_by_lot"`
	TrackBySerial   bool              `json:"track_by_serial"`
	TrackExpiration bool              `json:"track_expiration"`
	ImageURL        string            `json:"image_url"`
	MinQuantity     int               `json:"min_quantity"`
	MaxQuantity     int               `json:"max_quantity"`
	Lots            []database.Lot    `json:"lots"`
	Serials         []database.Serial `json:"serials"`
}
