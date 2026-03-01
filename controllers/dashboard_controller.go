package controllers

import (
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
	stats, response := c.Service.GetDashboardStats()

	if response != nil {
		writeErrorResponse(ctx, "GetDashboardStats", "get_dashboard_stats", response)
		return
	}

	tools.ResponseOK(ctx, "GetDashboardStats", "Estadísticas del dashboard obtenidas con éxito", "get_dashboard_stats", stats, false, "")
}
