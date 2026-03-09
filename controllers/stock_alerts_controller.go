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
		writeErrorResponse(ctx, "GetAllStockAlerts", "get_all_stock_alerts", response)
		return
	}

	if len(stockAlerts) == 0 {
		tools.ResponseOK(ctx, "GetAllStockAlerts", "No se encontraron alertas de stock", "get_all_stock_alerts", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllStockAlerts", "Alertas de stock obtenidas con éxito", "get_all_stock_alerts", stockAlerts, false, "")
}

func (c *StockAlertsController) Analyze(ctx *gin.Context) {
	responseData, response := c.Service.Analyze()

	if response != nil {
		writeErrorResponse(ctx, "Analyze", "analyze_stock_alerts", response)
		return
	}

	tools.ResponseOK(ctx, "Analyze", "Alertas de stock analizadas con éxito", "analyze_stock_alerts", responseData, false, "")
}

func (c *StockAlertsController) LotExpiration(ctx *gin.Context) {
	response, errResponse := c.Service.LotExpiration()
	if errResponse != nil {
		writeErrorResponse(ctx, "LotExpiration", "lot_expiration", errResponse)
		return
	}

	tools.ResponseOK(ctx, "LotExpiration", "Alertas de expiración de lotes generadas con éxito", "lot_expiration", response, false, "")
}

func (c *StockAlertsController) ResolveAlert(ctx *gin.Context) {
	alertID, ok := tools.ParseRequiredParam(ctx, "id", "ResolveAlert", "resolve_stock_alert", "ID de alerta inválido")
	if !ok {
		return
	}

	response := c.Service.ResolveAlert(alertID)

	if response != nil {
		writeErrorResponse(ctx, "ResolveAlert", "resolve_stock_alert", response)
		return
	}

	tools.ResponseOK(ctx, "ResolveAlert", "Alerta de stock resuelta con éxito", "resolve_stock_alert", nil, false, "")
}

func (c *StockAlertsController) ExportAlertsToExcel(ctx *gin.Context) {
	data, response := c.Service.ExportAlertsToExcel()

	if response != nil {
		writeErrorResponse(ctx, "ExportAlertsToExcel", "export_stock_alerts_to_excel", response)
		return
	}

	if data == nil {
		tools.ResponseOK(ctx, "ExportAlertsToExcel", "No se encontraron alertas de stock para exportar", "export_stock_alerts_to_excel", nil, false, "")
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=stock_alerts.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
