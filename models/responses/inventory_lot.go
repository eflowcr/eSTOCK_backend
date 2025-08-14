package responses

import "time"

type InventoryLot struct {
	ID             int
	InventoryID    int
	LotID          int
	Quantity       float64
	Location       string
	CreatedAt      time.Time
	LotNumber      string
	LotSKU         string
	LotQuantity    float64
	ExpirationDate *time.Time
	LotCreatedAt   time.Time
	LotUpdatedAt   time.Time
}
