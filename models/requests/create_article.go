package requests

type Article struct {
	SKU             string   `json:"sku" binding:"required" validate:"required,max=100"`
	Name            string   `json:"name" binding:"required" validate:"required,max=255"`
	Description     *string  `json:"description" validate:"omitempty,max=2000"`
	UnitPrice       *float64 `json:"unit_price" validate:"omitempty,gte=0"`
	Presentation    string   `json:"presentation" binding:"required" validate:"required,max=100"`
	TrackByLot       bool   `json:"track_by_lot"`
	TrackBySerial    bool   `json:"track_by_serial"`
	TrackExpiration  bool   `json:"track_expiration"`
	RotationStrategy string `json:"rotation_strategy" validate:"omitempty,oneof=fifo fefo"`
	MinQuantity      *int   `json:"min_quantity" validate:"omitempty,gte=0"`
	MaxQuantity     *int     `json:"max_quantity" validate:"omitempty,gte=0"`
	ImageURL        *string  `json:"image_url" validate:"omitempty,max=500"`
}
