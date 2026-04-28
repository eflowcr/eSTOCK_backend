package services

import (
	"context"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/rs/zerolog/log"
	stripe "github.com/stripe/stripe-go/v79"
	portalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
)

// BillingHandlerError wraps an InternalResponse with the event type that triggered it.
// Used by the webhook controller to log handler errors without stopping the HTTP response.
type BillingHandlerError struct {
	Resp      *responses.InternalResponse
	EventType string
}

// validPlans maps plan name → Stripe Price ID (populated at construction from config).
type BillingService struct {
	repo          ports.BillingRepository
	notifSvc      *NotificationsService
	tenantID      string
	priceIDs      map[string]string
	webhookSecret string
	appURL        string
}

// NewBillingService constructs a BillingService. Stripe API key is set globally (stripe.Key).
func NewBillingService(
	repo ports.BillingRepository,
	notifSvc *NotificationsService,
	tenantID string,
	cfg configuration.Config,
) *BillingService {
	stripe.Key = cfg.StripeSecretKey

	return &BillingService{
		repo:     repo,
		notifSvc: notifSvc,
		tenantID: tenantID,
		priceIDs: map[string]string{
			"starter":    cfg.StripePriceStarter,
			"pro":        cfg.StripePricePro,
			"enterprise": cfg.StripePriceEnterprise,
		},
		webhookSecret: cfg.StripeWebhookSecret,
		appURL:        cfg.AppURL,
	}
}

// PriceIDForPlan returns the Stripe Price ID for the given plan name, or an error if unknown.
func (s *BillingService) PriceIDForPlan(plan string) (string, error) {
	id, ok := s.priceIDs[plan]
	if !ok || id == "" {
		return "", fmt.Errorf("plan %q no configurado en STRIPE_PRICE_* env vars", plan)
	}
	return id, nil
}

// GetOrCreateStripeCustomer looks up the tenant's Stripe customer ID from the subscription record.
// If none exists, creates a new Stripe Customer and stores the ID.
func (s *BillingService) GetOrCreateStripeCustomer(tenantID, tenantEmail, tenantName string) (string, *responses.InternalResponse) {
	sub, resp := s.repo.GetSubscriptionByTenant(tenantID)
	if resp != nil {
		return "", resp
	}

	if sub != nil && sub.StripeCustomerID != nil && *sub.StripeCustomerID != "" {
		return *sub.StripeCustomerID, nil
	}

	// Create a new Stripe customer.
	params := &stripe.CustomerParams{
		Email: stripe.String(tenantEmail),
		Name:  stripe.String(tenantName),
		Metadata: map[string]string{
			"tenant_id": tenantID,
		},
	}
	cust, err := customer.New(params)
	if err != nil {
		return "", &responses.InternalResponse{
			Error:      err,
			Message:    "Error creando cliente en Stripe",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		}
	}

	// Persist the customer ID.
	if resp := s.repo.UpdateStripeCustomerID(tenantID, cust.ID); resp != nil {
		log.Warn().Err(resp.Error).Str("tenant_id", tenantID).Str("stripe_customer_id", cust.ID).
			Msg("billing: created Stripe customer but failed to persist ID")
	}

	return cust.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for the given plan and returns the URL.
func (s *BillingService) CreateCheckoutSession(tenantID, customerID, plan, priceID string) (string, *responses.InternalResponse) {
	successURL := s.appURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := s.appURL + "/billing/cancel"

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": tenantID,
				"plan":      plan,
			},
		},
		Metadata: map[string]string{
			"tenant_id": tenantID,
			"plan":      plan,
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", &responses.InternalResponse{
			Error:      err,
			Message:    "Error creando sesión de pago en Stripe",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		}
	}

	return sess.URL, nil
}

// CreatePortalSession creates a Stripe Billing Portal session for a customer and returns the URL.
func (s *BillingService) CreatePortalSession(customerID string) (string, *responses.InternalResponse) {
	returnURL := s.appURL + "/billing"

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return "", &responses.InternalResponse{
			Error:      err,
			Message:    "Error creando sesión del portal de facturación",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		}
	}

	return sess.URL, nil
}

// GetSubscription returns the current subscription for the given tenant.
//
// S3.5 W3: signature now requires tenantID (sourced from JWT claim by the controller).
// Previously read s.tenantID (env var) which made the BillingService unsafe in any pod
// serving more than one tenant. The injected s.tenantID is kept as a fallback ONLY for
// callers that pass "" (e.g. cron / system jobs); production endpoints MUST pass the
// per-request tenant.
func (s *BillingService) GetSubscription(tenantID string) (*database.Subscription, *responses.InternalResponse) {
	if tenantID == "" {
		tenantID = s.tenantID
	}
	return s.repo.GetSubscriptionByTenant(tenantID)
}

// GetTenantTrialInfo returns the tenant's trial_ends_at + status for the billing endpoint.
// Used by GET /api/billing/subscription to surface the trial deadline on the frontend banner
// even when there is no Stripe subscription yet (B4 fix — S3.5.5).
//
// Returns (nil, nil) if the tenant row does not exist (defensive — should never happen for an
// authenticated request, but the controller falls back gracefully).
func (s *BillingService) GetTenantTrialInfo(tenantID string) (*database.Tenant, *responses.InternalResponse) {
	if tenantID == "" {
		tenantID = s.tenantID
	}
	return s.repo.GetTenantByID(tenantID)
}

// HandleCheckoutSessionCompleted processes a checkout.session.completed Stripe event.
// Upserts the subscription record and sets the tenant status to active.
func (s *BillingService) HandleCheckoutSessionCompleted(sess *stripe.CheckoutSession) *responses.InternalResponse {
	tenantID := sess.Metadata["tenant_id"]
	plan := sess.Metadata["plan"]
	if tenantID == "" {
		log.Warn().Str("session_id", sess.ID).Msg("billing: checkout.session.completed missing tenant_id metadata")
		return nil
	}

	customerID := ""
	if sess.Customer != nil {
		customerID = sess.Customer.ID
	}
	subID := ""
	if sess.Subscription != nil {
		subID = sess.Subscription.ID
	}

	sub := &database.Subscription{
		TenantID: tenantID,
		Plan:     plan,
		Status:   "active",
	}
	if customerID != "" {
		sub.StripeCustomerID = &customerID
	}
	if subID != "" {
		sub.StripeSubscriptionID = &subID
	}
	now := time.Now()
	sub.CurrentPeriodStart = &now
	// TODO(CS3 — S3.5): CurrentPeriodEnd is never set from checkout.session.completed (period end
	// is on the subscription object, not the session). Initial billing record will show null
	// current_period_end until customer.subscription.updated fires. Consider fetching the
	// subscription from Stripe API here to populate CurrentPeriodEnd immediately.

	if resp := s.repo.UpsertSubscription(sub); resp != nil {
		return resp
	}

	if resp := s.repo.UpdateTenantStatus(tenantID, "active"); resp != nil {
		log.Warn().Err(resp.Error).Str("tenant_id", tenantID).Msg("billing: failed to update tenant status to active")
	}

	log.Info().Str("tenant_id", tenantID).Str("plan", plan).Str("stripe_sub_id", subID).
		Msg("billing: checkout session completed — subscription activated")
	return nil
}

// HandleSubscriptionUpdated processes a customer.subscription.updated event.
func (s *BillingService) HandleSubscriptionUpdated(stripeSub *stripe.Subscription) *responses.InternalResponse {
	tenantID := stripeSub.Metadata["tenant_id"]
	plan := stripeSub.Metadata["plan"]

	status := string(stripeSub.Status)
	cancelAtPeriodEnd := stripeSub.CancelAtPeriodEnd

	start := time.Unix(stripeSub.CurrentPeriodStart, 0)
	end := time.Unix(stripeSub.CurrentPeriodEnd, 0)

	update := &database.Subscription{
		Plan:               plan,
		CurrentPeriodStart: &start,
		CurrentPeriodEnd:   &end,
	}

	if resp := s.repo.UpdateSubscriptionStatus(stripeSub.ID, status, cancelAtPeriodEnd, update); resp != nil {
		return resp
	}

	// Keep tenant status in sync.
	if tenantID != "" {
		tenantStatus := status
		if status == "active" || status == "trialing" {
			tenantStatus = "active"
		}
		if resp := s.repo.UpdateTenantStatus(tenantID, tenantStatus); resp != nil {
			log.Warn().Err(resp.Error).Str("tenant_id", tenantID).Msg("billing: failed to sync tenant status on subscription update")
		}
	}

	log.Info().Str("stripe_sub_id", stripeSub.ID).Str("status", status).
		Bool("cancel_at_period_end", cancelAtPeriodEnd).
		Msg("billing: subscription updated")
	return nil
}

// HandleSubscriptionDeleted processes a customer.subscription.deleted event.
func (s *BillingService) HandleSubscriptionDeleted(stripeSub *stripe.Subscription) *responses.InternalResponse {
	tenantID := stripeSub.Metadata["tenant_id"]

	if resp := s.repo.UpdateSubscriptionStatus(stripeSub.ID, "cancelled", false, nil); resp != nil {
		return resp
	}

	if tenantID != "" {
		if resp := s.repo.UpdateTenantStatus(tenantID, "cancelled"); resp != nil {
			log.Warn().Err(resp.Error).Str("tenant_id", tenantID).Msg("billing: failed to mark tenant cancelled")
		}
	}

	log.Info().Str("stripe_sub_id", stripeSub.ID).Msg("billing: subscription deleted — status set to cancelled")
	return nil
}

// AttemptMarkWebhookEventProcessed atomically gates webhook idempotency.
// Returns alreadyProcessed=true if this event ID was already inserted — caller must skip.
func (s *BillingService) AttemptMarkWebhookEventProcessed(eventID, eventType string) (bool, *responses.InternalResponse) {
	return s.repo.AttemptMarkWebhookEventProcessed(eventID, eventType)
}

// IsWebhookEventProcessed checks idempotency via the billing repo.
func (s *BillingService) IsWebhookEventProcessed(eventID string) (bool, *responses.InternalResponse) {
	return s.repo.IsWebhookEventProcessed(eventID)
}

// MarkWebhookEventProcessed persists the event ID to prevent duplicate processing.
func (s *BillingService) MarkWebhookEventProcessed(eventID string) *responses.InternalResponse {
	return s.repo.MarkWebhookEventProcessed(eventID)
}

// HandleInvoicePaymentFailed processes an invoice.payment_failed event.
// Marks the subscription past_due and alerts the tenant admin.
//
// TODO(CS4 — S3.5): inv.Subscription may be an ID-only object (not expanded) in some Stripe API
// versions. If so, Metadata will be empty and tenantID remains "" — tenant status is never updated.
// Fix: if tenantID == "", look up via DB: SELECT tenant_id FROM subscriptions WHERE
// stripe_subscription_id = inv.Subscription.ID. Deferred to S3.5.
func (s *BillingService) HandleInvoicePaymentFailed(inv *stripe.Invoice) *responses.InternalResponse {
	tenantID := ""
	if inv.Subscription != nil {
		tenantID = inv.Subscription.Metadata["tenant_id"]
	}

	if inv.Subscription != nil {
		if resp := s.repo.UpdateSubscriptionStatus(inv.Subscription.ID, "past_due", false, nil); resp != nil {
			return resp
		}
	}

	if tenantID != "" {
		if resp := s.repo.UpdateTenantStatus(tenantID, "past_due"); resp != nil {
			log.Warn().Err(resp.Error).Str("tenant_id", tenantID).Msg("billing: failed to mark tenant past_due")
		}
	}

	// Notify tenant admin.
	if tenantID != "" && s.notifSvc != nil {
		adminID, resp := s.repo.GetTenantAdminUserID(tenantID)
		if resp == nil && adminID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.notifSvc.Send(ctx, adminID,
				"payment_failed",
				"Pago fallido — acción requerida",
				"Tu pago más reciente falló. Por favor actualiza tu método de pago para evitar la suspensión del servicio.",
				"subscription", tenantID,
			)
		}
	}

	log.Warn().Str("invoice_id", inv.ID).Str("tenant_id", tenantID).Msg("billing: invoice payment failed — subscription marked past_due")
	return nil
}
