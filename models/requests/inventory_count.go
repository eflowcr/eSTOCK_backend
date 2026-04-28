package requests

import "time"

// CreateInventoryCount is the payload for creating a new count sheet (admin/operator).
// Locations may be specified by id; the service expands them into rows in inventory_count_locations.
type CreateInventoryCount struct {
	Code         string     `json:"code" validate:"required,max=50"`
	Name         string     `json:"name" validate:"required,max=200"`
	Description  string     `json:"description" validate:"max=2000"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	LocationIDs  []string   `json:"location_ids" validate:"required,min=1,dive,required"`
}

// ScanCountLine is the body for POST /api/mobile/counts/:id/scan-line.
// Either SKU or Barcode must be provided; if only Barcode is set, the service resolves the SKU
// via the article catalog (multi-barcode support — TODO when barcode column lands).
type ScanCountLine struct {
	LocationID string  `json:"location_id" validate:"required"`
	SKU        string  `json:"sku" validate:"max=100"`
	Barcode    string  `json:"barcode" validate:"max=100"`
	Lot        string  `json:"lot" validate:"max=100"`
	Serial     string  `json:"serial" validate:"max=100"`
	ScannedQty float64 `json:"scanned_qty" validate:"required,gte=0"`
	Note       string  `json:"note" validate:"max=1000"`
}
