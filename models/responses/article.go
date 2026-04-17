package responses

import "time"

// EmbeddedCategory is a minimal category view embedded in ArticleResponse.
type EmbeddedCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EmbeddedLocation is a minimal location view embedded in ArticleResponse.
type EmbeddedLocation struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

// ArticleResponse is the enriched article view returned by GetArticleByID, UpdateArticle.
// It embeds optional category and default_location objects for consumers that need them.
type ArticleResponse struct {
	ID               string    `json:"id"`
	SKU              string    `json:"sku"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	UnitPrice        *float64  `json:"unit_price,omitempty"`
	Presentation     string    `json:"presentation"`
	TrackByLot       bool      `json:"track_by_lot"`
	TrackBySerial    bool      `json:"track_by_serial"`
	TrackExpiration  bool      `json:"track_expiration"`
	RotationStrategy string    `json:"rotation_strategy"`
	MinQuantity      *int      `json:"min_quantity,omitempty"`
	MaxQuantity      *int      `json:"max_quantity,omitempty"`
	ImageURL         *string   `json:"image_url,omitempty"`
	IsActive         *bool     `json:"is_active,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	// M2 extended fields
	CategoryID         *string           `json:"category_id,omitempty"`
	Category           *EmbeddedCategory `json:"category,omitempty"`
	ShelfLifeInDays    *int              `json:"shelf_life_in_days,omitempty"`
	SafetyStock        float64           `json:"safety_stock"`
	BatchNumberSeries  *string           `json:"batch_number_series,omitempty"`
	SerialNumberSeries *string           `json:"serial_number_series,omitempty"`
	MinOrderQty        float64           `json:"min_order_qty"`
	DefaultLocationID  *string           `json:"default_location_id,omitempty"`
	DefaultLocation    *EmbeddedLocation `json:"default_location,omitempty"`
	ReceivingNotes     *string           `json:"receiving_notes,omitempty"`
	ShippingNotes      *string           `json:"shipping_notes,omitempty"`
}
