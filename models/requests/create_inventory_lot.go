package requests

type CreateInventoryLotRequest struct {
	InventoryID int     `json:"inventoryId" binding:"required"`
	LotID       int     `json:"lotId" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	Location    string  `json:"location" binding:"required"`
}
