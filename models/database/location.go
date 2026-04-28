package database

import "time"

// Location is the master-data row for a warehouse storage point.
//
// S3.5 W2-A: tenant_id added so locations are tenant-scoped. The composite
// UNIQUE (tenant_id, location_code) lives in migration 000032 — the previous
// global UNIQUE on location_code was dropped.
type Location struct {
	ID           string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID     string    `gorm:"column:tenant_id;type:uuid;not null;index" json:"tenant_id"`
	LocationCode string    `gorm:"column:location_code" json:"location_code"`
	Description  *string   `gorm:"column:description" json:"description"`
	Zone         *string   `gorm:"column:zone" json:"zone"`
	Type         string    `gorm:"column:type" json:"type"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	IsWayOut     bool      `gorm:"column:is_way_out" json:"is_way_out"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Location) TableName() string {
	return "locations"
}
