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
type BillingController struct {
	Service       *services.BillingService
	TenantID      string
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

	// Get or create Stripe customer. Use tenant email/name from JWT claims if available,
	// or fall back to tenant ID as display name (frontend can update later).
	tenantEmail, _ := ctx.Get("email")
	tenantEmailStr, _ := tenantEmail.(string)
	if tenantEmailStr == "" {
		tenantEmailStr = c.TenantID + "@tenant.estock.app"
	}

	customerID, resp := c.Service.GetOrCreateStripeCustomer(c.TenantID, tenantEmailStr, c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingCheckout", "billing_checkout", resp)
		return
	}

	checkoutURL, resp := c.Service.CreateCheckoutSession(c.TenantID, customerID, req.Plan, priceID)
	if resp != nil {
		writeErrorResponse(ctx, "BillingCheckout", "billing_checkout", resp)
		return
	}

	tools.ResponseOK(ctx, "BillingCheckout", "Sesión de checkout creada", "billing_checkout",
		gin.H{"url": checkoutURL}, false, "")
}

// GetSubscription handles GET /api/billing/subscription (JWT required, tenant-scoped).
// Returns the current subscription record for the tenant.
func (c *BillingController) GetSubscription(ctx *gin.Context) {
	sub, resp := c.Service.GetSubscription()
	if resp != nil {
		writeErrorResponse(ctx, "GetBillingSubscription", "get_billing_subscription", resp)
		return
	}
	if sub == nil {
		tools.ResponseOK(ctx, "GetBillingSubscription", "Sin suscripción activa", "get_billing_subscription",
			gin.H{"subscription": nil}, false, "")
		return
	}
	tools.ResponseOK(ctx, "GetBillingSubscription", "Suscripción obtenida", "get_billing_subscription",
		gin.H{"subscription": sub}, false, "")
}

// PortalSession handles POST /api/billing/portal-session (JWT required, tenant-scoped).
// Creates a Stripe Billing Portal session and returns the URL.
func (c *BillingController) PortalSession(ctx *gin.Context) {
	sub, resp := c.Service.GetSubscription()
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

	// Idempotency: skip already-processed events.
	processed, iresp := c.Service.IsWebhookEventProcessed(event.ID)
	if iresp != nil {
		log.Warn().Err(iresp.Error).Str("event_id", event.ID).Msg("billing webhook: idempotency check failed — processing anyway")
	} else if processed {
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

	// Mark event as processed (best-effort — don't fail the webhook if this errors).
	if resp := c.Service.MarkWebhookEventProcessed(event.ID); resp != nil {
		log.Warn().Err(resp.Error).Str("event_id", event.ID).Msg("billing webhook: failed to mark event as processed")
	}

	ctx.Status(http.StatusOK)
}

