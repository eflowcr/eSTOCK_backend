package requests

type CreateInventorySerial struct {
	InventoryID string `json:"inventoryId" validate:"required"`
	SerialID    string `json:"serialId" validate:"required"`
	Location    string `json:"location" validate:"required,max=100"`
}
