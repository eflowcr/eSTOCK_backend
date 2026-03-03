package requests

type CreateInventorySerial struct {
	InventoryID int    `json:"inventoryId" validate:"required,gt=0"`
	SerialID    int    `json:"serialId" validate:"required,gt=0"`
	Location    string `json:"location" validate:"required,max=100"`
}
