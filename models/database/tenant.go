package database

import (
	"encoding/json"
	"time"
)

// Tenant represents a SaaS tenant in the shared-DB multi-tenant model.
// Status: trial|active|past_due|cancelled|suspended.
// The default tenant (id=00000000-0000-0000-0000-000000000001) is backfilled
// by migration 000023 and represents legacy single-tenant deployments.
type Tenant struct {
	ID              string          `gorm:"column:id;primaryKey" json:"id"`
	Name            string          `gorm:"column:name" json:"name"`
	Slug            string          `gorm:"column:slug;unique" json:"slug"`
	Email           string          `gorm:"column:email" json:"email"`
	Status          string          `gorm:"column:status" json:"status"` // trial|active|past_due|cancelled|suspended
	SignupAt         time.Time       `gorm:"column:signup_at" json:"signup_at"`
	TrialStartedAt  time.Time       `gorm:"column:trial_started_at" json:"trial_started_at"`
	TrialEndsAt     time.Time       `gorm:"column:trial_ends_at" json:"trial_ends_at"`
	IsActive        bool            `gorm:"column:is_active" json:"is_active"`
	Metadata        json.RawMessage `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       *time.Time      `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (Tenant) TableName() string {
	return "tenants"
}
