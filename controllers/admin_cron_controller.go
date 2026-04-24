package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminCronController struct {
	DB *gorm.DB
}

func NewAdminCronController(db *gorm.DB) *AdminCronController {
	return &AdminCronController{DB: db}
}

// Trigger handles POST /admin/cron/trigger?job=stock_alerts|stale_reservations|trial_expiration|all
// Protected by JWTAuthMiddleware + RequirePermission("cron","trigger").
func (c *AdminCronController) Trigger(ctx *gin.Context) {
	job := ctx.DefaultQuery("job", "all")

	analyzer := func() error {
		repo := &repositories.StockAlertsRepository{DB: c.DB}
		svc := services.NewStockAlertsService(repo)
		if _, resp := svc.Analyze(); resp != nil && resp.Error != nil {
			return resp.Error
		}
		if _, resp := svc.LotExpiration(); resp != nil && resp.Error != nil {
			return resp.Error
		}
		return nil
	}

	switch job {
	case "stock_alerts":
		if err := tools.RunStockAlertAnalysis(c.DB, analyzer); err != nil {
			tools.ResponseInternal(ctx, "CronTrigger", "Error al ejecutar stock_alerts", "cron_trigger")
			return
		}
	case "stale_reservations":
		if err := tools.RunStaleReservationsCleanup(c.DB); err != nil {
			tools.ResponseInternal(ctx, "CronTrigger", "Error al ejecutar stale_reservations", "cron_trigger")
			return
		}
	case "trial_expiration":
		// Admin manual trigger for trial lifecycle check. Email sends are fire-and-forget (nil sendFn
		// skips emails) — callers that need emails should wire a real sendFn via the background cron.
		if err := tools.RunTrialExpirationCheck(c.DB, nil); err != nil {
			tools.ResponseInternal(ctx, "CronTrigger", "Error al ejecutar trial_expiration", "cron_trigger")
			return
		}
	case "all":
		// Admin manual trigger: no notification callbacks (fire-and-forget; notifications
		// are wired in the background cron goroutine in main.go).
		tools.CronDispatch(c.DB, analyzer, nil, nil, nil)
	default:
		tools.ResponseBadRequest(ctx, "CronTrigger", "Job inválido. Use: stock_alerts | stale_reservations | trial_expiration | all", "cron_trigger")
		return
	}

	tools.ResponseOK(ctx, "CronTrigger", "Job ejecutado", "cron_trigger", gin.H{"job": job}, false, "")
}
