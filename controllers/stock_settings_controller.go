package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type StockSettingsController struct {
	Service  services.StockSettingsService
	TenantID string
}

func NewStockSettingsController(service services.StockSettingsService, tenantID string) *StockSettingsController {
	return &StockSettingsController{Service: service, TenantID: tenantID}
}

func (c *StockSettingsController) Get(ctx *gin.Context) {
	settings, resp := c.Service.GetOrCreate(c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "GetStockSettings", "get_stock_settings", resp)
		return
	}
	tools.ResponseOK(ctx, "GetStockSettings", "Configuración de stock recuperada", "get_stock_settings", settings, false, "")
}

func (c *StockSettingsController) Update(ctx *gin.Context) {
	var req requests.UpdateStockSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateStockSettings", "Datos de solicitud inválidos", "update_stock_settings")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateStockSettings", "update_stock_settings", errs)
		return
	}

	settings, resp := c.Service.Update(c.resolveTenantID(ctx), &req)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateStockSettings", "update_stock_settings", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateStockSettings", "Configuración actualizada", "update_stock_settings", settings, false, "")
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *StockSettingsController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
