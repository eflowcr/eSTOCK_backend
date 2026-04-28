package routes

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ ports.SignupRepository = (*repositories.SignupRepository)(nil)

// RegisterSignupRoutes registers the public signup endpoints under /api.
// Both routes are public (no auth middleware) but have aggressive IP rate limits
// to prevent abuse:
//
//   POST /api/signup        — 5 per hour per IP
//   POST /api/signup/verify — 10 per hour per IP
//
// rolesRepo is optional; when supplied the verify response is enriched with the
// admin role name + permissions so the frontend's auto-login produces a fully
// hydrated session (S3.5.6 B22). When nil, behavior degrades gracefully and the
// frontend recovers via logout+login.
func RegisterSignupRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	repo := &repositories.SignupRepository{
		DB:          db,
		Config:      config,
		EmailSender: wire.EmailSenderForConfig(config),
	}
	svc := services.NewSignupService(repo, rolesRepo)
	ctrl := controllers.NewSignupController(svc)

	signup := router.Group("/signup")
	{
		// 5 requests per hour per IP (one per 12 minutes)
		signup.POST("",
			tools.NewIPRateLimiter(rate.Every(12*time.Minute), 5),
			ctrl.InitiateSignup)

		// 10 requests per hour per IP (one per 6 minutes)
		signup.POST("/verify",
			tools.NewIPRateLimiter(rate.Every(6*time.Minute), 10),
			ctrl.VerifySignup)
	}
}
