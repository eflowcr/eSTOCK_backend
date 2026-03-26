package requests

// ArticleImportRow represents a single article row sent from the frontend preview table.
type ArticleImportRow struct {
	SKU              string `json:"sku"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	UnitPrice        string `json:"unit_price"`
	Presentation     string `json:"presentation"`
	TrackByLot       string `json:"track_by_lot"`
	TrackBySerial    string `json:"track_by_serial"`
	TrackExpiration  string `json:"track_expiration"`
	MaxQuantity      string `json:"max_quantity"`
	MinQuantity      string `json:"min_quantity"`
	RotationStrategy string `json:"rotation_strategy"`
}
