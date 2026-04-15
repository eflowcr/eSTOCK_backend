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

// Trigger handles POST /admin/cron/trigger?job=stock_alerts|stale_reservations|all
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
	case "all":
		tools.CronDispatch(c.DB, analyzer)
	default:
		tools.ResponseBadRequest(ctx, "CronTrigger", "Job inválido. Use: stock_alerts | stale_reservations | all", "cron_trigger")
		return
	}

	tools.ResponseOK(ctx, "CronTrigger", "Job ejecutado", "cron_trigger", gin.H{"job": job}, false, "")
}
