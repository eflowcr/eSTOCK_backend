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
		tools.Response(ctx, "GetDashboardStats", false, response.Message, "get_dashboard_stats", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "GetDashboardStats", true, "Dashboard stats retrieved successfully", "get_dashboard_stats", stats, false, "", false)
}
