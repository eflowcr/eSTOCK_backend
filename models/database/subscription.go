package database

import (
	"encoding/json"
	"time"
)

// Subscription tracks a tenant's Stripe-backed billing subscription.
// Plan: trial|starter|pro|enterprise.
// Status mirrors Stripe subscription status: active|past_due|cancelled|incomplete|trialing.
type Subscription struct {
	ID                   string          `gorm:"column:id;primaryKey" json:"id"`
	TenantID             string          `gorm:"column:tenant_id" json:"tenant_id"`
	StripeSubscriptionID *string         `gorm:"column:stripe_subscription_id;unique" json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     *string         `gorm:"column:stripe_customer_id" json:"stripe_customer_id,omitempty"`
	Plan                 string          `gorm:"column:plan" json:"plan"`
	Status               string          `gorm:"column:status" json:"status"`
	CurrentPeriodStart   *time.Time      `gorm:"column:current_period_start" json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time      `gorm:"column:current_period_end" json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool            `gorm:"column:cancel_at_period_end" json:"cancel_at_period_end"`
	TrialEnd             *time.Time      `gorm:"column:trial_end" json:"trial_end,omitempty"`
	Metadata             json.RawMessage `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt            time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
