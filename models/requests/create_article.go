package requests

type Article struct {
	SKU              string   `json:"sku" binding:"required" validate:"required,max=100"`
	Name             string   `json:"name" binding:"required" validate:"required,max=255"`
	Description      *string  `json:"description" validate:"omitempty,max=2000"`
	UnitPrice        *float64 `json:"unit_price" validate:"omitempty,gte=0"`
	Presentation     string   `json:"presentation" binding:"required" validate:"required,max=100"`
	TrackByLot       bool     `json:"track_by_lot"`
	TrackBySerial    bool     `json:"track_by_serial"`
	TrackExpiration  bool     `json:"track_expiration"`
	RotationStrategy string   `json:"rotation_strategy" validate:"omitempty,oneof=fifo fefo"`
	MinQuantity      *int     `json:"min_quantity" validate:"omitempty,gte=0"`
	MaxQuantity      *int     `json:"max_quantity" validate:"omitempty,gte=0"`
	ImageURL         *string  `json:"image_url" validate:"omitempty,max=500"`
	// M2 extended fields
	CategoryID         *string  `json:"category_id,omitempty"`
	ShelfLifeInDays    *int     `json:"shelf_life_in_days,omitempty" validate:"omitempty,gte=0"`
	SafetyStock        float64  `json:"safety_stock" validate:"gte=0"`
	BatchNumberSeries  *string  `json:"batch_number_series,omitempty" validate:"omitempty,max=50"`
	SerialNumberSeries *string  `json:"serial_number_series,omitempty" validate:"omitempty,max=50"`
	MinOrderQty        float64  `json:"min_order_qty" validate:"gte=0"`
	DefaultLocationID  *string  `json:"default_location_id,omitempty"`
	ReceivingNotes     *string  `json:"receiving_notes,omitempty"`
	ShippingNotes      *string  `json:"shipping_notes,omitempty"`
}
