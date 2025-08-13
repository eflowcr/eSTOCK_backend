package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type StockAlertsController struct {
	Service services.StockAlertsService
}

func NewStockAlertsController(service services.StockAlertsService) *StockAlertsController {
	return &StockAlertsController{
		Service: service,
	}
}

func (c *StockAlertsController) GetAllStockAlerts(ctx *gin.Context) {
	resolved := ctx.Param("resolved") == "true"
	stockAlerts, response := c.Service.GetAllStockAlerts(resolved)

	if response != nil {
		tools.Response(ctx, "GetAllStockAlerts", false, response.Message, "get_all_stock_alerts", nil, false, "")
		return
	}

	if len(stockAlerts) == 0 {
		tools.Response(ctx, "GetAllStockAlerts", true, "No stock alerts found", "get_all_stock_alerts", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllStockAlerts", true, "Stock alerts retrieved successfully", "get_all_stock_alerts", stockAlerts, false, "")
}

func (c *StockAlertsController) Analyze(ctx *gin.Context) {
	responseData, response := c.Service.Analyze()

	if response != nil {
		tools.Response(ctx, "Analyze", false, response.Message, "analyze_stock_alerts", nil, false, "")
		return
	}

	tools.Response(ctx, "Analyze", true, "Stock alerts analyzed successfully", "analyze_stock_alerts", responseData, false, "")
}
