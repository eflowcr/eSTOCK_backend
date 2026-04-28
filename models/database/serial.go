package database

import "time"

// Serial is the master-data row for a serial-tracked unit.
//
// S3.5 W2-A: tenant_id added so serials are tenant-scoped. The composite
// UNIQUE (tenant_id, serial_number) lives in migration 000033.
type Serial struct {
	ID           string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID     string    `gorm:"column:tenant_id;type:uuid;not null;index" json:"tenant_id"`
	SerialNumber string    `gorm:"column:serial_number" json:"serial_number"`
	SKU          string    `gorm:"column:sku" json:"sku"`
	Status       string    `gorm:"column:status" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Serial) TableName() string {
	return "serials"
}
