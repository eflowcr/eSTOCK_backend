package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// BackordersController handles HTTP for backorder endpoints (BO2).
type BackordersController struct {
	Service  *services.BackordersService
	TenantID string
}

func NewBackordersController(svc *services.BackordersService, tenantID string) *BackordersController {
	return &BackordersController{Service: svc, TenantID: tenantID}
}

// List handles GET /api/backorders/
func (c *BackordersController) List(ctx *gin.Context) {
	var status, soID *string

	if v := ctx.Query("status"); v != "" {
		status = &v
	}
	if v := ctx.Query("so_id"); v != "" {
		soID = &v
	}

	page := 1
	limit := 50
	if p := ctx.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	result, resp := c.Service.List(c.TenantID, status, soID, page, limit)
	if resp != nil {
		writeErrorResponse(ctx, "ListBackorders", "list_backorders", resp)
		return
	}
	tools.ResponseOK(ctx, "ListBackorders", "Backorders recuperados", "list_backorders", result, false, "")
}

// GetByID handles GET /api/backorders/:id
func (c *BackordersController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetBackorder", "get_backorder", "ID de backorder inválido")
	if !ok {
		return
	}

	bo, resp := c.Service.GetByID(id, c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "GetBackorder", "get_backorder", resp)
		return
	}
	tools.ResponseOK(ctx, "GetBackorder", "Backorder recuperado", "get_backorder", bo, false, "")
}

// Fulfill handles POST /api/backorders/:id/fulfill
// Creates a new picking task from pending backorder stock (BO2).
func (c *BackordersController) Fulfill(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "FulfillBackorder", "fulfill_backorder", "ID de backorder inválido")
	if !ok {
		return
	}

	userID := ctx.GetString(tools.ContextKeyUserID)

	result, resp := c.Service.Fulfill(id, c.TenantID, userID)
	if resp != nil {
		writeErrorResponse(ctx, "FulfillBackorder", "fulfill_backorder", resp)
		return
	}
	tools.ResponseCreated(ctx, "FulfillBackorder", "Tarea de picking creada para fulfilliar backorder", "fulfill_backorder", result, false, "")
}
