package requests

type CreateInventoryLotRequest struct {
	InventoryID int     `json:"inventoryId"`
	LotID       int     `json:"lotId" validate:"required,gt=0"`
	Quantity    float64 `json:"quantity" validate:"required,gte=0"`
	Location    string  `json:"location" validate:"required"`
}
