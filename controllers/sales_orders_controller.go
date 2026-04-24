package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// SalesOrdersController handles HTTP for /api/sales-orders.
type SalesOrdersController struct {
	Service   *services.SalesOrdersService
	JWTSecret string
	TenantID  string
}

func NewSalesOrdersController(svc *services.SalesOrdersService, jwtSecret, tenantID string) *SalesOrdersController {
	return &SalesOrdersController{Service: svc, JWTSecret: jwtSecret, TenantID: tenantID}
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — CRUD
// ─────────────────────────────────────────────────────────────────────────────

// List godoc: GET /api/sales-orders
func (c *SalesOrdersController) List(ctx *gin.Context) {
	page, limit, ok := tools.ParseQueryPageLimit(ctx, "ListSalesOrders", "list_sales_orders")
	if !ok {
		return
	}

	var status, customerID, search, dateFrom, dateTo *string
	if v := ctx.Query("status"); v != "" {
		status = &v
	}
	if v := ctx.Query("customer_id"); v != "" {
		customerID = &v
	}
	if v := ctx.Query("search"); v != "" {
		search = &v
	}
	if v := ctx.Query("date_from"); v != "" {
		dateFrom = &v
	}
	if v := ctx.Query("date_to"); v != "" {
		dateTo = &v
	}

	result, resp := c.Service.List(c.TenantID, status, customerID, search, dateFrom, dateTo, page, limit)
	if resp != nil {
		writeErrorResponse(ctx, "ListSalesOrders", "list_sales_orders", resp)
		return
	}
	tools.ResponseOK(ctx, "ListSalesOrders", "Órdenes de venta recuperadas", "list_sales_orders", result, false, "")
}

// GetByID godoc: GET /api/sales-orders/:id
func (c *SalesOrdersController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetSalesOrder", "get_sales_order", "ID de orden inválido")
	if !ok {
		return
	}
	so, resp := c.Service.GetByID(id, c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "GetSalesOrder", "get_sales_order", resp)
		return
	}
	tools.ResponseOK(ctx, "GetSalesOrder", "Orden de venta recuperada", "get_sales_order", so, false, "")
}

// Create godoc: POST /api/sales-orders
func (c *SalesOrdersController) Create(ctx *gin.Context) {
	var req requests.CreateSalesOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "CreateSalesOrder", "Datos de solicitud inválidos", "create_sales_order")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "CreateSalesOrder", "create_sales_order", errs)
		return
	}

	userID, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "CreateSalesOrder", "Token inválido", "create_sales_order")
		return
	}

	so, resp := c.Service.Create(c.TenantID, userID, &req)
	if resp != nil {
		writeErrorResponse(ctx, "CreateSalesOrder", "create_sales_order", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateSalesOrder", "Orden de venta creada", "create_sales_order", so, false, "")
}

// Update godoc: PATCH /api/sales-orders/:id
func (c *SalesOrdersController) Update(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateSalesOrder", "update_sales_order", "ID de orden inválido")
	if !ok {
		return
	}
	var req requests.UpdateSalesOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateSalesOrder", "Datos de solicitud inválidos", "update_sales_order")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateSalesOrder", "update_sales_order", errs)
		return
	}

	so, resp := c.Service.Update(id, c.TenantID, &req)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateSalesOrder", "update_sales_order", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateSalesOrder", "Orden de venta actualizada", "update_sales_order", so, false, "")
}

// SoftDelete godoc: DELETE /api/sales-orders/:id
func (c *SalesOrdersController) SoftDelete(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteSalesOrder", "delete_sales_order", "ID de orden inválido")
	if !ok {
		return
	}
	if resp := c.Service.SoftDelete(id, c.TenantID); resp != nil {
		writeErrorResponse(ctx, "DeleteSalesOrder", "delete_sales_order", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteSalesOrder", "Orden de venta eliminada", "delete_sales_order", nil, false, "")
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — Lifecycle
// ─────────────────────────────────────────────────────────────────────────────

// Submit godoc: PATCH /api/sales-orders/:id/submit
func (c *SalesOrdersController) Submit(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "SubmitSalesOrder", "submit_sales_order", "ID de orden inválido")
	if !ok {
		return
	}
	userID, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "SubmitSalesOrder", "Token inválido", "submit_sales_order")
		return
	}

	result, resp := c.Service.Submit(id, c.TenantID, userID)
	if resp != nil {
		writeErrorResponse(ctx, "SubmitSalesOrder", "submit_sales_order", resp)
		return
	}
	tools.ResponseOK(ctx, "SubmitSalesOrder", "Orden enviada — tarea de picking generada", "submit_sales_order", result, false, "")
}

// Cancel godoc: PATCH /api/sales-orders/:id/cancel
func (c *SalesOrdersController) Cancel(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CancelSalesOrder", "cancel_sales_order", "ID de orden inválido")
	if !ok {
		return
	}
	userID, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "CancelSalesOrder", "Token inválido", "cancel_sales_order")
		return
	}

	if resp := c.Service.Cancel(id, c.TenantID, userID); resp != nil {
		writeErrorResponse(ctx, "CancelSalesOrder", "cancel_sales_order", resp)
		return
	}
	tools.ResponseOK(ctx, "CancelSalesOrder", "Orden de venta cancelada", "cancel_sales_order", nil, false, "")
}
