package repositories

import (
	"errors"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BillingRepository implements ports.BillingRepository using GORM.
type BillingRepository struct {
	DB *gorm.DB
}

var _ ports.BillingRepository = (*BillingRepository)(nil)

// GetSubscriptionByTenant returns the current subscription for a tenant, or nil if none exists.
func (r *BillingRepository) GetSubscriptionByTenant(tenantID string) (*database.Subscription, *responses.InternalResponse) {
	var sub database.Subscription
	err := r.DB.Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error obteniendo suscripción", Handled: false}
	}
	return &sub, nil
}

// UpsertSubscription creates or updates a subscription. Uses stripe_subscription_id as the conflict key
// when present; otherwise falls back to a plain Create (for initial checkout.session.completed before
// the Stripe subscription ID is known).
func (r *BillingRepository) UpsertSubscription(sub *database.Subscription) *responses.InternalResponse {
	if sub.ID == "" {
		id, err := tools.GenerateNanoid(r.DB)
		if err != nil {
			return &responses.InternalResponse{Error: err, Message: "Error generando ID de suscripción", Handled: false}
		}
		sub.ID = id
	}

	if sub.StripeSubscriptionID != nil && *sub.StripeSubscriptionID != "" {
		if err := r.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stripe_subscription_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"stripe_customer_id",
				"plan",
				"status",
				"current_period_start",
				"current_period_end",
				"cancel_at_period_end",
				"trial_end",
				"metadata",
				"updated_at",
			}),
		}).Create(sub).Error; err != nil {
			return &responses.InternalResponse{Error: err, Message: "Error guardando suscripción", Handled: false}
		}
		return nil
	}

	// No stripe_subscription_id yet — upsert by tenant_id.
	if err := r.DB.Clauses(clause.OnConflict{
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: "stripe_subscription_id", Value: nil},
		}},
	}).FirstOrCreate(sub, "tenant_id = ?", sub.TenantID).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error guardando suscripción inicial", Handled: false}
	}
	return nil
}

// UpdateSubscriptionStatus updates status, periods, and cancel_at_period_end for a subscription
// identified by stripeSubscriptionID. All non-zero fields from sub are applied.
func (r *BillingRepository) UpdateSubscriptionStatus(stripeSubscriptionID, status string, cancelAtPeriodEnd bool, sub *database.Subscription) *responses.InternalResponse {
	updates := map[string]interface{}{
		"status":               status,
		"cancel_at_period_end": cancelAtPeriodEnd,
		"updated_at":           time.Now(),
	}
	if sub != nil {
		if sub.CurrentPeriodStart != nil {
			updates["current_period_start"] = sub.CurrentPeriodStart
		}
		if sub.CurrentPeriodEnd != nil {
			updates["current_period_end"] = sub.CurrentPeriodEnd
		}
		if sub.Plan != "" {
			updates["plan"] = sub.Plan
		}
	}

	result := r.DB.Model(&database.Subscription{}).
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		Updates(updates)
	if result.Error != nil {
		return &responses.InternalResponse{Error: result.Error, Message: "Error actualizando estado de suscripción", Handled: false}
	}
	return nil
}

// UpdateStripeCustomerID stores the Stripe customer ID on the tenant's subscription record.
func (r *BillingRepository) UpdateStripeCustomerID(tenantID, stripeCustomerID string) *responses.InternalResponse {
	if err := r.DB.Model(&database.Subscription{}).
		Where("tenant_id = ?", tenantID).
		Update("stripe_customer_id", stripeCustomerID).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error guardando stripe_customer_id", Handled: false}
	}
	return nil
}

// UpdateTenantStatus updates the tenant's status field.
func (r *BillingRepository) UpdateTenantStatus(tenantID, status string) *responses.InternalResponse {
	if err := r.DB.Model(&database.Tenant{}).
		Where("id = ?", tenantID).
		Updates(map[string]interface{}{
			"status":     status,
			"is_active":  status == "active",
			"updated_at": time.Now(),
		}).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error actualizando estado del tenant", Handled: false}
	}
	return nil
}

// AttemptMarkWebhookEventProcessed atomically gates webhook idempotency via INSERT ON CONFLICT.
// It attempts to insert the event_id as a PRIMARY KEY. If the event was already inserted (i.e.
// a concurrent or retry delivery), RowsAffected will be 0 and alreadyProcessed=true is returned —
// the caller must skip processing. This is a single round-trip with no TOCTOU window.
func (r *BillingRepository) AttemptMarkWebhookEventProcessed(eventID, eventType string) (alreadyProcessed bool, resp *responses.InternalResponse) {
	result := r.DB.Exec(
		"INSERT INTO stripe_webhook_events (event_id, event_type, processed_at) VALUES (?, ?, NOW()) ON CONFLICT (event_id) DO NOTHING",
		eventID, eventType,
	)
	if result.Error != nil {
		return false, &responses.InternalResponse{Error: result.Error, Message: "Error registrando evento de webhook", Handled: false}
	}
	// RowsAffected == 0 → conflict → already processed.
	return result.RowsAffected == 0, nil
}

// IsWebhookEventProcessed checks if the event ID has been processed (idempotency via stripe_webhook_events table).
func (r *BillingRepository) IsWebhookEventProcessed(eventID string) (bool, *responses.InternalResponse) {
	var count int64
	if err := r.DB.Table("stripe_webhook_events").
		Where("event_id = ?", eventID).
		Count(&count).Error; err != nil {
		return false, &responses.InternalResponse{Error: err, Message: "Error verificando evento de webhook", Handled: false}
	}
	return count > 0, nil
}

// MarkWebhookEventProcessed records a Stripe event ID as processed (best-effort).
// Deprecated: prefer AttemptMarkWebhookEventProcessed which is atomic and eliminates TOCTOU.
func (r *BillingRepository) MarkWebhookEventProcessed(eventID string) *responses.InternalResponse {
	if err := r.DB.Exec(
		"INSERT INTO stripe_webhook_events (event_id, event_type, processed_at) VALUES (?, '', NOW()) ON CONFLICT (event_id) DO NOTHING",
		eventID,
	).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error registrando evento de webhook", Handled: false}
	}
	return nil
}

// GetTenantAdminUserID returns the user ID of the first admin for a tenant (for notifications).
func (r *BillingRepository) GetTenantAdminUserID(tenantID string) (string, *responses.InternalResponse) {
	var userID string
	if err := r.DB.Table("users").
		Select("id").
		Where("tenant_id = ? AND role = 'admin' AND deleted_at IS NULL", tenantID).
		Order("created_at ASC").
		Limit(1).
		Scan(&userID).Error; err != nil {
		return "", &responses.InternalResponse{Error: err, Message: "Error buscando admin del tenant", Handled: false}
	}
	return userID, nil
}
