package database

import (
	"encoding/json"
	"time"
)

// DemoDataSeed tracks which demo data sets have been seeded for a tenant.
// SeedName examples: 'farma-50skus', 'retail-20skus'.
// Metadata captures counts (articles, locations, tasks) at seed time.
type DemoDataSeed struct {
	ID       string          `gorm:"column:id;primaryKey" json:"id"`
	TenantID string          `gorm:"column:tenant_id" json:"tenant_id"`
	SeedName string          `gorm:"column:seed_name" json:"seed_name"`
	SeededAt time.Time       `gorm:"column:seeded_at;autoCreateTime" json:"seeded_at"`
	Metadata json.RawMessage `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
}

func (DemoDataSeed) TableName() string {
	return "demo_data_seeds"
}
