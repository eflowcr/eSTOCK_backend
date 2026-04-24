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

	// IsWebhookEventProcessed checks whether a Stripe event ID has already been processed (idempotency).
	IsWebhookEventProcessed(eventID string) (bool, *responses.InternalResponse)

	// MarkWebhookEventProcessed records a Stripe event ID to prevent duplicate processing.
	MarkWebhookEventProcessed(eventID string) *responses.InternalResponse

	// GetTenantAdminUserID returns the user ID of the admin for a tenant (for notifications).
	GetTenantAdminUserID(tenantID string) (string, *responses.InternalResponse)
}
