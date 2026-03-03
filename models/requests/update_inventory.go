package requests

type UpdateInventory struct {
	SKU                string   `json:"sku" validate:"required,max=100"`
	Name               string   `json:"name" validate:"required,max=255"`
	Description        *string  `json:"description" validate:"omitempty,max=2000"`
	Location           string   `json:"location" validate:"required,max=100"`
	Quantity           float64  `json:"quantity" validate:"required,gte=0"`
	UnitPrice          *float64 `json:"unit_price" validate:"required,gte=0"`
	DefaultLotNumber   *string  `json:"defaultLotNumber,omitempty" validate:"omitempty,max=100"`
	SerialNumberPrefix *string  `json:"serialNumberPrefix,omitempty" validate:"omitempty,max=100"`
	Status             string   `json:"status" validate:"max=50"`
	TrackByLot         bool     `json:"trackByLot"`
	TrackBySerial      bool     `json:"trackBySerial"`
}
