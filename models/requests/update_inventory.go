package requests

type UpdateInventory struct {
	SKU                string   `json:"sku" validate:"required"`
	Name               string   `json:"name" validate:"required"`
	Description        *string  `json:"description"`
	Location           string   `json:"location" validate:"required"`
	Quantity           float64  `json:"quantity" validate:"required"`
	UnitPrice          *float64 `json:"unit_price" validate:"required"`
	DefaultLotNumber   *string  `json:"defaultLotNumber,omitempty"`
	SerialNumberPrefix *string  `json:"serialNumberPrefix,omitempty"`
	TrackByLot         bool     `json:"trackByLot"`
	TrackBySerial      bool     `json:"trackBySerial"`
}
