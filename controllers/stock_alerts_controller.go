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
		tools.Response(ctx, "GetAllStockAlerts", false, response.Message, "get_all_stock_alerts", nil, false, "", response.Handled)
		return
	}

	if len(stockAlerts) == 0 {
		tools.Response(ctx, "GetAllStockAlerts", true, "No stock alerts found", "get_all_stock_alerts", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllStockAlerts", true, "Stock alerts retrieved successfully", "get_all_stock_alerts", stockAlerts, false, "", false)
}

func (c *StockAlertsController) Analyze(ctx *gin.Context) {
	responseData, response := c.Service.Analyze()

	if response != nil {
		tools.Response(ctx, "Analyze", false, response.Message, "analyze_stock_alerts", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "Analyze", true, "Stock alerts analyzed successfully", "analyze_stock_alerts", responseData, false, "", false)
}

func (c *StockAlertsController) LotExpiration(ctx *gin.Context) {
	response, errResponse := c.Service.LotExpiration()
	if errResponse != nil {
		tools.Response(ctx, "LotExpiration", false, errResponse.Message, "lot_expiration", nil, false, "", false)
		return
	}

	tools.Response(ctx, "LotExpiration", true, "Lot expiration alerts generated successfully", "lot_expiration", response, false, "", false)
}

func (c *StockAlertsController) ResolveAlert(ctx *gin.Context) {
	alertID := ctx.Param("id")

	alertIDInt, err := tools.StringToInt(alertID)

	if err != nil {
		tools.Response(ctx, "ResolveAlert", false, "Invalid alert ID", "resolve_stock_alert", nil, false, "", false)
		return
	}

	response := c.Service.ResolveAlert(alertIDInt)

	if response != nil {
		tools.Response(ctx, "Resolve", false, response.Message, "resolve_stock_alert", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "Resolve", true, "Stock alert resolved successfully", "resolve_stock_alert", nil, false, "", false)
}

func (c *StockAlertsController) ExportAlertsToExcel(ctx *gin.Context) {
	data, response := c.Service.ExportAlertsToExcel()

	if response != nil {
		tools.Response(ctx, "ExportAlertsToExcel", false, response.Message, "export_stock_alerts_to_excel", nil, false, "", response.Handled)
		return
	}

	if data == nil {
		tools.Response(ctx, "ExportAlertsToExcel", true, "No stock alerts to export", "export_stock_alerts_to_excel", nil, false, "", false)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=stock_alerts.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
