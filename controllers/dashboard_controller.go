package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	Service services.DashboardService
}

func NewDashboardController(service services.DashboardService) *DashboardController {
	return &DashboardController{
		Service: service,
	}
}

func (c *DashboardController) GetDashboardStats(ctx *gin.Context) {
	tasksPeriod := ctx.DefaultQuery("tasksPeriod", "weekly")
	lowStockThreshold := 20
	if v := ctx.Query("lowStockThreshold"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			lowStockThreshold = n
		}
	}
	stats, response := c.Service.GetDashboardStats(tasksPeriod, lowStockThreshold)

	if response != nil {
		writeErrorResponse(ctx, "GetDashboardStats", "get_dashboard_stats", response)
		return
	}

	tools.ResponseOK(ctx, "GetDashboardStats", "Estadísticas del dashboard obtenidas con éxito", "get_dashboard_stats", stats, false, "")
}

func (c *DashboardController) GetInventorySummary(ctx *gin.Context) {
	period := ctx.DefaultQuery("period", "monthly")
	summary, response := c.Service.GetInventorySummary(period)

	if response != nil {
		writeErrorResponse(ctx, "GetInventorySummary", "get_inventory_summary", response)
		return
	}

	tools.ResponseOK(ctx, "GetInventorySummary", "Resumen de inventario obtenido con éxito", "get_inventory_summary", summary, false, "")
}

func (c *DashboardController) GetMovementsMonthly(ctx *gin.Context) {
	period := ctx.DefaultQuery("period", "monthly")
	data, response := c.Service.GetMovementsMonthly(period)

	if response != nil {
		writeErrorResponse(ctx, "GetMovementsMonthly", "get_movements_monthly", response)
		return
	}

	tools.ResponseOK(ctx, "GetMovementsMonthly", "Movimientos mensuales obtenidos con éxito", "get_movements_monthly", data, false, "")
}

func (c *DashboardController) GetRecentActivity(ctx *gin.Context) {
	data, response := c.Service.GetRecentActivity()

	if response != nil {
		writeErrorResponse(ctx, "GetRecentActivity", "get_recent_activity", response)
		return
	}

	tools.ResponseOK(ctx, "GetRecentActivity", "Actividad reciente obtenida con éxito", "get_recent_activity", data, false, "")
}
