package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// PurchaseOrdersController handles HTTP for purchase order endpoints (PO1 + PO2).
type PurchaseOrdersController struct {
	Service  *services.PurchaseOrdersService
	TenantID string
}

func NewPurchaseOrdersController(svc *services.PurchaseOrdersService, tenantID string) *PurchaseOrdersController {
	return &PurchaseOrdersController{Service: svc, TenantID: tenantID}
}

// ─────────────────────────────────────────────────────────────────────────────
// PO1 — CRUD
// ─────────────────────────────────────────────────────────────────────────────

// Create handles POST /api/purchase-orders/
func (c *PurchaseOrdersController) Create(ctx *gin.Context) {
	var req requests.CreatePurchaseOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "CreatePurchaseOrder", "Datos de solicitud inválidos", "create_purchase_order")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "CreatePurchaseOrder", "create_purchase_order", errs)
		return
	}

	userID := ctx.GetString(tools.ContextKeyUserID)

	po, resp := c.Service.Create(c.resolveTenantID(ctx), userID, &req)
	if resp != nil {
		writeErrorResponse(ctx, "CreatePurchaseOrder", "create_purchase_order", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreatePurchaseOrder", "Orden de compra creada exitosamente", "create_purchase_order", po, false, "")
}

// List handles GET /api/purchase-orders/
func (c *PurchaseOrdersController) List(ctx *gin.Context) {
	var status, supplierID, search, from, to *string

	if v := ctx.Query("status"); v != "" {
		status = &v
	}
	if v := ctx.Query("supplier_id"); v != "" {
		supplierID = &v
	}
	if v := ctx.Query("search"); v != "" {
		search = &v
	}
	if v := ctx.Query("from"); v != "" {
		from = &v
	}
	if v := ctx.Query("to"); v != "" {
		to = &v
	}

	limit := 50
	offset := 0
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := ctx.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	pos, resp := c.Service.List(c.resolveTenantID(ctx), status, supplierID, search, from, to, limit, offset)
	if resp != nil {
		writeErrorResponse(ctx, "ListPurchaseOrders", "list_purchase_orders", resp)
		return
	}
	tools.ResponseOK(ctx, "ListPurchaseOrders", "Órdenes de compra recuperadas", "list_purchase_orders", pos, false, "")
}

// GetByID handles GET /api/purchase-orders/:id
func (c *PurchaseOrdersController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetPurchaseOrder", "get_purchase_order", "ID de orden de compra inválido")
	if !ok {
		return
	}

	po, resp := c.Service.GetByID(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "GetPurchaseOrder", "get_purchase_order", resp)
		return
	}
	tools.ResponseOK(ctx, "GetPurchaseOrder", "Orden de compra recuperada", "get_purchase_order", po, false, "")
}

// Update handles PATCH /api/purchase-orders/:id
func (c *PurchaseOrdersController) Update(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdatePurchaseOrder", "update_purchase_order", "ID de orden de compra inválido")
	if !ok {
		return
	}

	var req requests.UpdatePurchaseOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePurchaseOrder", "Datos de solicitud inválidos", "update_purchase_order")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdatePurchaseOrder", "update_purchase_order", errs)
		return
	}

	po, resp := c.Service.Update(id, c.resolveTenantID(ctx), &req)
	if resp != nil {
		writeErrorResponse(ctx, "UpdatePurchaseOrder", "update_purchase_order", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdatePurchaseOrder", "Orden de compra actualizada", "update_purchase_order", po, false, "")
}

// Delete handles DELETE /api/purchase-orders/:id
func (c *PurchaseOrdersController) Delete(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeletePurchaseOrder", "delete_purchase_order", "ID de orden de compra inválido")
	if !ok {
		return
	}

	if resp := c.Service.SoftDelete(id, c.resolveTenantID(ctx)); resp != nil {
		writeErrorResponse(ctx, "DeletePurchaseOrder", "delete_purchase_order", resp)
		return
	}
	tools.ResponseOK(ctx, "DeletePurchaseOrder", "Orden de compra eliminada", "delete_purchase_order", nil, false, "")
}

// ─────────────────────────────────────────────────────────────────────────────
// PO2 — Lifecycle
// ─────────────────────────────────────────────────────────────────────────────

// Submit handles PATCH /api/purchase-orders/:id/submit
func (c *PurchaseOrdersController) Submit(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "SubmitPurchaseOrder", "submit_purchase_order", "ID de orden de compra inválido")
	if !ok {
		return
	}

	userID := ctx.GetString(tools.ContextKeyUserID)

	po, newRTID, resp := c.Service.Submit(id, c.resolveTenantID(ctx), userID)
	if resp != nil {
		writeErrorResponse(ctx, "SubmitPurchaseOrder", "submit_purchase_order", resp)
		return
	}

	// Return PO + new_receiving_task_id in a single payload.
	type submitPayload struct {
		PurchaseOrder      interface{} `json:"purchase_order"`
		NewReceivingTaskID string      `json:"new_receiving_task_id"`
	}
	tools.ResponseOK(ctx, "SubmitPurchaseOrder", "Orden de compra sometida y tarea de recepción generada", "submit_purchase_order", submitPayload{
		PurchaseOrder:      po,
		NewReceivingTaskID: newRTID,
	}, false, "")
}

// Cancel handles PATCH /api/purchase-orders/:id/cancel
func (c *PurchaseOrdersController) Cancel(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CancelPurchaseOrder", "cancel_purchase_order", "ID de orden de compra inválido")
	if !ok {
		return
	}

	po, resp := c.Service.Cancel(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "CancelPurchaseOrder", "cancel_purchase_order", resp)
		return
	}
	tools.ResponseOK(ctx, "CancelPurchaseOrder", "Orden de compra cancelada", "cancel_purchase_order", po, false, "")
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *PurchaseOrdersController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
