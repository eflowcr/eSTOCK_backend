package database

import "time"

// InventoryLot is the junction row linking inventory ↔ lots with per-location
// quantity.
//
// S3.5 W2-A: tenant_id added so junction writes are tenant-scoped. The
// composite UNIQUE (tenant_id, inventory_id, lot_id, location) prevents
// duplicate per-tenant allocations (migration 000034).
type InventoryLot struct {
	ID          string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID    string    `gorm:"column:tenant_id;type:uuid;not null;index" json:"tenant_id"`
	InventoryID string    `gorm:"column:inventory_id" json:"inventory_id"`
	LotID       string    `gorm:"column:lot_id" json:"lot_id"`
	Quantity    float64   `gorm:"column:quantity" json:"quantity"`
	Location    string    `gorm:"column:location" json:"location"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventoryLot) TableName() string {
	return "inventory_lots"
}
