package requests

type CreateInventorySerial struct {
	InventoryID int    `json:"inventoryId" validate:"required"`
	SerialID    int    `json:"serialId" validate:"required"`
	Location    string `json:"location" validate:"required"`
}
