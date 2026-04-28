package controllers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	stripe "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
)

// BillingController handles Stripe billing endpoints.
//
// S3.5 W3: tenant is sourced per-request from the JWT claim (TenantIDFromContext) instead
// of a constructor-injected env var. The TenantID field is kept ONLY as a default-only
// fallback for system / cron / admin paths that bypass JWTAuthMiddleware — never used by
// the JWT-protected endpoints.
type BillingController struct {
	Service       *services.BillingService
	TenantID      string // fallback for non-JWT callers only (cron, admin tooling)
	WebhookSecret string
}

// NewBillingController constructs a BillingController.
func NewBillingController(svc *services.BillingService, tenantID, webhookSecret string) *BillingController {
	return &BillingController{
		Service:       svc,
		TenantID:      tenantID,
		WebhookSecret: webhookSecret,
	}
}

// resolveTenantID returns the tenant for this request: JWT claim first, env-var fallback only
// if the claim is missing AND the controller has a default (system path). Returns "" iff there
// is no tenant available — the caller MUST then return 401 to avoid leaking another tenant's
// data via Config.TenantID.
func (c *BillingController) resolveTenantID(ctx *gin.Context) string {
	if t := tools.TenantIDFromContext(ctx); t != "" {
		return t
	}
	return c.TenantID
}

// Checkout handles POST /api/billing/checkout (JWT required, tenant-scoped).
// Body: {"plan": "starter"|"pro"|"enterprise"}
// Returns: {"url": "<stripe_checkout_url>"}
func (c *BillingController) Checkout(ctx *gin.Context) {
	var req requests.CheckoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "BillingCheckout", "Datos de solicitud inválidos", "billing_checkout")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "BillingCheckout", "billing_checkout", errs)
		return
	}

	priceID, err := c.Service.PriceIDForPlan(req.Plan)
	if err != nil {
		tools.ResponseBadRequest(ctx, "BillingCheckout", err.Error(), "billing_checkout")
		return
	}

	// S3.5 W3 — read tenant from JWT claim, never from c.TenantID (env var) for JWT-protected endpoints.
	tenantID := c.resolveTenantID(ctx)
	if tenantID == "" {
		tools.ResponseUnauthorized(ctx, "BillingCheckout", "tenant no identificado en token", "billing_checkout")
		return
	}

	// Get or create Stripe customer. Use tenant email/name from JWT claims if available,
	// or fall back to tenant ID as display name (frontend can update later).
	tenantEmail, _ := ctx.Get("email")
	tenantEmailStr, _ := tenantEmail.(string)
	if tenantEmailStr == "" {
		tenantEmailStr = tenantID + "@tenant.estock.app"
	}

	// TODO(CS2 — S3.5): third arg is tenantName but passes tenantID (a UUID). Stripe dashboard
	// shows UUID as customer name — unidentifiable. Should use tenant's company name from DB or JWT.
	customerID, resp := c.Service.GetOrCreateStripeCustomer(tenantID, tenantEmailStr, tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingCheckout", "billing_checkout", resp)
		return
	}

	checkoutURL, resp := c.Service.CreateCheckoutSession(tenantID, customerID, req.Plan, priceID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingCheckout", "billing_checkout", resp)
		return
	}

	tools.ResponseOK(ctx, "BillingCheckout", "Sesión de checkout creada", "billing_checkout",
		gin.H{"url": checkoutURL}, false, "")
}

// GetSubscription handles GET /api/billing/subscription (JWT required, tenant-scoped).
// Returns the current subscription record for the tenant, plus trial_ends_at + tenant_status
// from the tenants table so the frontend banner can show the real trial deadline regardless of
// whether a Stripe subscription exists yet (B4 fix — S3.5.5).
//
// Response shape:
//
//	{
//	  "data": {
//	    "subscription": <Subscription | null>,
//	    "trial_ends_at": "2026-05-11T20:07:37Z" | null,
//	    "tenant_status": "trial" | "active" | ...
//	  }
//	}
//
// trial_ends_at is sourced from tenants.trial_ends_at (NOT subscription.trial_end) because the
// trial period is defined at the tenant level by the signup flow — Stripe's trial_end only
// applies once a paid plan with a trial is selected.
func (c *BillingController) GetSubscription(ctx *gin.Context) {
	tenantID := c.resolveTenantID(ctx)
	if tenantID == "" {
		tools.ResponseUnauthorized(ctx, "GetBillingSubscription", "tenant no identificado en token", "get_billing_subscription")
		return
	}
	sub, resp := c.Service.GetSubscription(tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "GetBillingSubscription", "get_billing_subscription", resp)
		return
	}

	// Fetch tenant trial info regardless of whether a subscription exists. The frontend
	// uses tenants.trial_ends_at as the source of truth for the trial banner.
	// If the tenant lookup fails we log + degrade (return subscription only) — the banner
	// will fall back to "expires today" but the subscription endpoint stays functional.
	tenant, tresp := c.Service.GetTenantTrialInfo(tenantID)
	var trialEndsAt *string
	tenantStatus := ""
	if tresp != nil {
		log.Warn().Err(tresp.Error).Str("tenant_id", tenantID).
			Msg("billing: failed to load tenant trial info — degrading response without trial_ends_at")
	} else if tenant != nil {
		tenantStatus = tenant.Status
		if !tenant.TrialEndsAt.IsZero() {
			s := tenant.TrialEndsAt.UTC().Format("2006-01-02T15:04:05Z")
			trialEndsAt = &s
		}
	}

	payload := gin.H{
		"subscription":  sub,
		"trial_ends_at": trialEndsAt,
		"tenant_status": tenantStatus,
	}

	if sub == nil {
		tools.ResponseOK(ctx, "GetBillingSubscription", "Sin suscripción activa", "get_billing_subscription",
			payload, false, "")
		return
	}
	tools.ResponseOK(ctx, "GetBillingSubscription", "Suscripción obtenida", "get_billing_subscription",
		payload, false, "")
}

// PortalSession handles POST /api/billing/portal-session (JWT required, tenant-scoped).
// Creates a Stripe Billing Portal session and returns the URL.
func (c *BillingController) PortalSession(ctx *gin.Context) {
	tenantID := c.resolveTenantID(ctx)
	if tenantID == "" {
		tools.ResponseUnauthorized(ctx, "BillingPortalSession", "tenant no identificado en token", "billing_portal_session")
		return
	}
	sub, resp := c.Service.GetSubscription(tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingPortalSession", "billing_portal_session", resp)
		return
	}
	if sub == nil || sub.StripeCustomerID == nil || *sub.StripeCustomerID == "" {
		tools.ResponseBadRequest(ctx, "BillingPortalSession",
			"No hay suscripción de Stripe asociada a este tenant. Completa el proceso de checkout primero.",
			"billing_portal_session")
		return
	}

	portalURL, resp := c.Service.CreatePortalSession(*sub.StripeCustomerID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingPortalSession", "billing_portal_session", resp)
		return
	}

	tools.ResponseOK(ctx, "BillingPortalSession", "Sesión del portal creada", "billing_portal_session",
		gin.H{"url": portalURL}, false, "")
}

// StripeWebhook handles POST /api/billing/stripe-webhook (NO JWT — verified by Stripe signature).
// Verifies the webhook signature and dispatches event types.
func (c *BillingController) StripeWebhook(ctx *gin.Context) {
	const maxBodyBytes = int64(65536)
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)

	payload, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Warn().Err(err).Msg("billing webhook: failed to read body")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	sigHeader := ctx.GetHeader("Stripe-Signature")
	if sigHeader == "" {
		log.Warn().Msg("billing webhook: missing Stripe-Signature header")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Verify signature — OBLIGATORIO. Never process unverified events.
	event, err := webhook.ConstructEvent(payload, sigHeader, c.WebhookSecret)
	if err != nil {
		log.Warn().Err(err).Msg("billing webhook: signature verification failed")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Idempotency gate: atomically INSERT event_id (PRIMARY KEY). If RowsAffected==0, another
	// delivery already claimed this event — return 200 immediately without re-processing.
	// This single round-trip eliminates the TOCTOU race of a separate SELECT + INSERT.
	alreadyProcessed, iresp := c.Service.AttemptMarkWebhookEventProcessed(event.ID, string(event.Type))
	if iresp != nil {
		log.Error().Err(iresp.Error).Str("event_id", event.ID).Msg("billing webhook: idempotency gate failed — aborting to avoid double-processing")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	if alreadyProcessed {
		log.Debug().Str("event_id", event.ID).Str("type", string(event.Type)).Msg("billing webhook: duplicate event — skipping")
		ctx.Status(http.StatusOK)
		return
	}

	// Dispatch by event type.
	var handlerErr *services.BillingHandlerError
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Error().Err(err).Str("event_id", event.ID).Msg("billing webhook: failed to parse checkout.session.completed")
			ctx.Status(http.StatusOK) // Return 200 so Stripe doesn't retry a bad payload
			return
		}
		if resp := c.Service.HandleCheckoutSessionCompleted(&sess); resp != nil {
			handlerErr = &services.BillingHandlerError{Resp: resp, EventType: string(event.Type)}
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Error().Err(err).Str("event_id", event.ID).Msg("billing webhook: failed to parse customer.subscription.updated")
			ctx.Status(http.StatusOK)
			return
		}
		if resp := c.Service.HandleSubscriptionUpdated(&sub); resp != nil {
			handlerErr = &services.BillingHandlerError{Resp: resp, EventType: string(event.Type)}
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Error().Err(err).Str("event_id", event.ID).Msg("billing webhook: failed to parse customer.subscription.deleted")
			ctx.Status(http.StatusOK)
			return
		}
		if resp := c.Service.HandleSubscriptionDeleted(&sub); resp != nil {
			handlerErr = &services.BillingHandlerError{Resp: resp, EventType: string(event.Type)}
		}

	case "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			log.Error().Err(err).Str("event_id", event.ID).Msg("billing webhook: failed to parse invoice.payment_failed")
			ctx.Status(http.StatusOK)
			return
		}
		if resp := c.Service.HandleInvoicePaymentFailed(&inv); resp != nil {
			handlerErr = &services.BillingHandlerError{Resp: resp, EventType: string(event.Type)}
		}

	default:
		log.Debug().Str("event_id", event.ID).Str("type", string(event.Type)).Msg("billing webhook: unhandled event type — ignoring")
	}

	if handlerErr != nil {
		log.Error().Err(handlerErr.Resp.Error).
			Str("event_id", event.ID).
			Str("event_type", handlerErr.EventType).
			Msg("billing webhook: handler error")
		// Return 200 so Stripe doesn't retry indefinitely for internal errors.
		// The event is logged for manual investigation.
		ctx.Status(http.StatusOK)
		return
	}

	// Event was marked as processed atomically at the start (AttemptMarkWebhookEventProcessed).
	// No second write needed here.

	ctx.Status(http.StatusOK)
}
