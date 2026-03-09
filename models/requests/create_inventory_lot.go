package requests

type CreateInventoryLotRequest struct {
	InventoryID string  `json:"inventoryId"`
	LotID       string  `json:"lotId" validate:"required"`
	Quantity    float64 `json:"quantity" validate:"required,gte=0"`
	Location    string  `json:"location" validate:"required,max=100"`
}
