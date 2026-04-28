package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterBillingRoutes mounts Stripe billing endpoints under /api/billing.
//
// JWT-protected:
//   POST /api/billing/checkout           — create Stripe Checkout Session
//   GET  /api/billing/subscription       — current subscription data
//   POST /api/billing/portal-session     — create Stripe Billing Portal session
//
// Stripe-signature-protected (NO JWT):
//   POST /api/billing/stripe-webhook     — Stripe webhook receiver
func RegisterBillingRoutes(api *gin.RouterGroup, db *gorm.DB, config configuration.Config, notifSvc *services.NotificationsService, rolesRepo ports.RolesRepository) {
	if db == nil {
		return
	}

	repo := &repositories.BillingRepository{DB: db}
	billingSvc := services.NewBillingService(repo, notifSvc, config.TenantID, config)
	ctrl := controllers.NewBillingController(billingSvc, config.TenantID, config.StripeWebhookSecret)

	billing := api.Group("/billing")

	// ── Stripe webhook — NO JWT, verified by Stripe signature ──────────────
	// Must be registered BEFORE the JWT middleware group so Stripe requests
	// don't hit the auth middleware.
	billing.POST("/stripe-webhook", ctrl.StripeWebhook)

	// ── JWT-protected billing endpoints ─────────────────────────────────────
	protected := billing.Group("")
	protected.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		protected.POST("/checkout", ctrl.Checkout)
		protected.GET("/subscription", ctrl.GetSubscription)
		protected.POST("/portal-session", ctrl.PortalSession)
	}
}
