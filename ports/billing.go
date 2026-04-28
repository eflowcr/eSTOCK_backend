package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// BillingRepository defines persistence operations for subscriptions and Stripe billing data.
type BillingRepository interface {
	// GetSubscriptionByTenant returns the current subscription for a tenant, or nil if none exists.
	GetSubscriptionByTenant(tenantID string) (*database.Subscription, *responses.InternalResponse)

	// UpsertSubscription creates or updates a subscription by stripe_subscription_id.
	// If stripeSubscriptionID is empty, upserts by tenant_id (for checkout.session.completed flows).
	UpsertSubscription(sub *database.Subscription) *responses.InternalResponse

	// UpdateSubscriptionStatus updates the status (and cancel_at_period_end, periods) of a subscription
	// identified by stripeSubscriptionID.
	UpdateSubscriptionStatus(stripeSubscriptionID, status string, cancelAtPeriodEnd bool, sub *database.Subscription) *responses.InternalResponse

	// UpdateStripeCustomerID stores the Stripe customer ID on the subscription record for the tenant.
	UpdateStripeCustomerID(tenantID, stripeCustomerID string) *responses.InternalResponse

	// UpdateTenantStatus updates the tenant's status field (trial|active|past_due|cancelled).
	UpdateTenantStatus(tenantID, status string) *responses.InternalResponse

	// AttemptMarkWebhookEventProcessed atomically inserts event_id into stripe_webhook_events
	// (INSERT … ON CONFLICT DO NOTHING). Returns alreadyProcessed=true if a duplicate — the
	// caller must skip processing without an error. This eliminates the TOCTOU race between a
	// separate SELECT check and INSERT mark.
	AttemptMarkWebhookEventProcessed(eventID, eventType string) (alreadyProcessed bool, resp *responses.InternalResponse)

	// IsWebhookEventProcessed checks whether a Stripe event ID has already been processed (idempotency).
	// Retained for service-layer unit tests and direct queries. Prefer AttemptMarkWebhookEventProcessed
	// for the actual webhook gate.
	IsWebhookEventProcessed(eventID string) (bool, *responses.InternalResponse)

	// MarkWebhookEventProcessed records a Stripe event ID as processed (best-effort after handling).
	// Deprecated: use AttemptMarkWebhookEventProcessed as the atomic gate instead.
	MarkWebhookEventProcessed(eventID string) *responses.InternalResponse

	// GetTenantAdminUserID returns the user ID of the admin for a tenant (for notifications).
	GetTenantAdminUserID(tenantID string) (string, *responses.InternalResponse)

	// GetTenantByID returns the tenant row for a tenant ID, or nil if not found.
	// Used by GET /api/billing/subscription to expose trial_ends_at + status when the
	// tenant has no Stripe subscription yet (B4 fix — S3.5.5).
	GetTenantByID(tenantID string) (*database.Tenant, *responses.InternalResponse)
}
