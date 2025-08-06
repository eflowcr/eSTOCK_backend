package requests

type Article struct {
	SKU             string   `json:"sku" binding:"required"`
	Name            string   `json:"name" binding:"required"`
	Description     *string  `json:"description"` // Opcional
	UnitPrice       *float64 `json:"unit_price"`  // Opcional
	Presentation    string   `json:"presentation" binding:"required"`
	TrackByLot      bool     `json:"track_by_lot"`
	TrackBySerial   bool     `json:"track_by_serial"`
	TrackExpiration bool     `json:"track_expiration"`
	MinQuantity     *int     `json:"min_quantity"` // Opcional
	MaxQuantity     *int     `json:"max_quantity"` // Opcional
	ImageURL        *string  `json:"image_url"`    // Opcional
}
