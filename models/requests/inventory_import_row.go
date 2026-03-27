package requests

// InventoryImportRow represents a single inventory row from the frontend preview table.
type InventoryImportRow struct {
	SKU             string `json:"sku"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Location        string `json:"location"`
	Quantity        string `json:"quantity"`
	UnitPrice       string `json:"unit_price"`
	TrackByLot      string `json:"track_by_lot"`
	TrackBySerial   string `json:"track_by_serial"`
	TrackExpiration string `json:"track_expiration"`
	MinQuantity     string `json:"min_quantity"`
	MaxQuantity     string `json:"max_quantity"`
}
