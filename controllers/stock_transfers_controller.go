package controllers

import (
	"encoding/json"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type StockTransfersController struct {
	Service     services.StockTransfersService
	JWTSecret   string
	AuditService *services.AuditService
}

func NewStockTransfersController(service services.StockTransfersService, jwtSecret string, auditSvc *services.AuditService) *StockTransfersController {
	return &StockTransfersController{Service: service, JWTSecret: jwtSecret, AuditService: auditSvc}
}

func (c *StockTransfersController) auditUserID(ctx *gin.Context) *string {
	userIDVal, _ := ctx.Get(tools.ContextKeyUserID)
	if idStr, ok := userIDVal.(string); ok && idStr != "" {
		return &idStr
	}
	return nil
}

func (c *StockTransfersController) ListStockTransfers(ctx *gin.Context) {
	status := ctx.Query("status")
	list, resp := c.Service.ListStockTransfers(status)
	if resp != nil {
		writeErrorResponse(ctx, "ListStockTransfers", "list_stock_transfers", resp)
		return
	}
	tools.ResponseOK(ctx, "ListStockTransfers", "Stock transfers retrieved", "list_stock_transfers", list, false, "")
}

func (c *StockTransfersController) GetStockTransferByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetStockTransferByID", "get_stock_transfer_by_id", "Invalid stock transfer ID")
	if !ok {
		return
	}
	transfer, resp := c.Service.GetStockTransferByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetStockTransferByID", "get_stock_transfer_by_id", resp)
		return
	}
	if transfer == nil {
		tools.ResponseNotFound(ctx, "GetStockTransferByID", "Stock transfer not found", "get_stock_transfer_by_id")
		return
	}
	tools.ResponseOK(ctx, "GetStockTransferByID", "Stock transfer retrieved", "get_stock_transfer_by_id", transfer, false, "")
}

func (c *StockTransfersController) CreateStockTransfer(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userID, _ := tools.GetUserId(c.JWTSecret, token)
	if userID == "" {
		tools.ResponseUnauthorized(ctx, "CreateStockTransfer", "Unauthorized", "create_stock_transfer")
		return
	}
	var body requests.StockTransferCreate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateStockTransfer", "Invalid request body", "create_stock_transfer")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateStockTransfer", "create_stock_transfer", errs)
		return
	}
	created, resp := c.Service.CreateStockTransfer(&body, userID)
	if resp != nil {
		writeErrorResponse(ctx, "CreateStockTransfer", "create_stock_transfer", resp)
		return
	}
	if c.AuditService != nil && created != nil {
		newVal, _ := json.Marshal(created)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionCreate, tools.ResourceStockTransfer, created.ID, nil, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseCreated(ctx, "CreateStockTransfer", "Stock transfer created", "create_stock_transfer", created, false, "")
}

func (c *StockTransfersController) UpdateStockTransfer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateStockTransfer", "update_stock_transfer", "Invalid stock transfer ID")
	if !ok {
		return
	}
	var body requests.StockTransferUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateStockTransfer", "Invalid request body", "update_stock_transfer")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateStockTransfer", "update_stock_transfer", errs)
		return
	}
	existing, _ := c.Service.GetStockTransferByID(id)
	updated, resp := c.Service.UpdateStockTransfer(id, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateStockTransfer", "update_stock_transfer", resp)
		return
	}
	if c.AuditService != nil && updated != nil {
		oldVal, _ := json.Marshal(existing)
		newVal, _ := json.Marshal(updated)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionUpdate, tools.ResourceStockTransfer, id, oldVal, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "UpdateStockTransfer", "Stock transfer updated", "update_stock_transfer", updated, false, "")
}

func (c *StockTransfersController) DeleteStockTransfer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteStockTransfer", "delete_stock_transfer", "Invalid stock transfer ID")
	if !ok {
		return
	}
	existing, _ := c.Service.GetStockTransferByID(id)
	resp := c.Service.DeleteStockTransfer(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeleteStockTransfer", "delete_stock_transfer", resp)
		return
	}
	if c.AuditService != nil && existing != nil {
		oldVal, _ := json.Marshal(existing)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionDelete, tools.ResourceStockTransfer, id, oldVal, nil, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "DeleteStockTransfer", "Stock transfer deleted", "delete_stock_transfer", nil, false, "")
}

func (c *StockTransfersController) ExecuteStockTransfer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "ExecuteStockTransfer", "execute_stock_transfer", "Invalid stock transfer ID")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, _ := tools.GetUserId(c.JWTSecret, token)
	if userID == "" {
		tools.ResponseUnauthorized(ctx, "ExecuteStockTransfer", "Unauthorized", "execute_stock_transfer")
		return
	}
	existing, _ := c.Service.GetStockTransferByID(id)
	transfer, resp := c.Service.ExecuteTransfer(id, userID)
	if resp != nil {
		writeErrorResponse(ctx, "ExecuteStockTransfer", "execute_stock_transfer", resp)
		return
	}
	if c.AuditService != nil && transfer != nil {
		oldVal, _ := json.Marshal(existing)
		newVal, _ := json.Marshal(transfer)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionExecute, tools.ResourceStockTransfer, id, oldVal, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "ExecuteStockTransfer", "Stock transfer executed", "execute_stock_transfer", transfer, false, "")
}

func (c *StockTransfersController) ListStockTransferLines(ctx *gin.Context) {
	transferID, ok := tools.ParseRequiredParam(ctx, "id", "ListStockTransferLines", "list_stock_transfer_lines", "Invalid stock transfer ID")
	if !ok {
		return
	}
	list, resp := c.Service.ListStockTransferLines(transferID)
	if resp != nil {
		writeErrorResponse(ctx, "ListStockTransferLines", "list_stock_transfer_lines", resp)
		return
	}
	tools.ResponseOK(ctx, "ListStockTransferLines", "Stock transfer lines retrieved", "list_stock_transfer_lines", list, false, "")
}

func (c *StockTransfersController) CreateStockTransferLine(ctx *gin.Context) {
	transferID, ok := tools.ParseRequiredParam(ctx, "id", "CreateStockTransferLine", "create_stock_transfer_line", "Invalid stock transfer ID")
	if !ok {
		return
	}
	var body requests.StockTransferLineInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateStockTransferLine", "Invalid request body", "create_stock_transfer_line")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateStockTransferLine", "create_stock_transfer_line", errs)
		return
	}
	created, resp := c.Service.CreateStockTransferLine(transferID, &body)
	if resp != nil {
		writeErrorResponse(ctx, "CreateStockTransferLine", "create_stock_transfer_line", resp)
		return
	}
	if c.AuditService != nil && created != nil {
		newVal, _ := json.Marshal(created)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionCreate, tools.ResourceStockTransfer, transferID, nil, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseCreated(ctx, "CreateStockTransferLine", "Stock transfer line created", "create_stock_transfer_line", created, false, "")
}

func (c *StockTransfersController) UpdateStockTransferLine(ctx *gin.Context) {
	lineID, ok := tools.ParseRequiredParam(ctx, "lineId", "UpdateStockTransferLine", "update_stock_transfer_line", "Invalid line ID")
	if !ok {
		return
	}
	var body requests.StockTransferLineUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateStockTransferLine", "Invalid request body", "update_stock_transfer_line")
		return
	}
	transferID, _ := ctx.Params.Get("id")
	updated, resp := c.Service.UpdateStockTransferLine(lineID, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateStockTransferLine", "update_stock_transfer_line", resp)
		return
	}
	if c.AuditService != nil && updated != nil {
		newVal, _ := json.Marshal(updated)
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionUpdate, tools.ResourceStockTransfer, transferID, nil, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "UpdateStockTransferLine", "Stock transfer line updated", "update_stock_transfer_line", updated, false, "")
}

func (c *StockTransfersController) DeleteStockTransferLine(ctx *gin.Context) {
	lineID, ok := tools.ParseRequiredParam(ctx, "lineId", "DeleteStockTransferLine", "delete_stock_transfer_line", "Invalid line ID")
	if !ok {
		return
	}
	transferID, _ := ctx.Params.Get("id")
	resp := c.Service.DeleteStockTransferLine(lineID)
	if resp != nil {
		writeErrorResponse(ctx, "DeleteStockTransferLine", "delete_stock_transfer_line", resp)
		return
	}
	if c.AuditService != nil {
		oldVal, _ := json.Marshal(map[string]string{"line_id": lineID, "transfer_id": transferID})
		c.AuditService.Log(ctx.Request.Context(), c.auditUserID(ctx), tools.ActionDelete, tools.ResourceStockTransfer, transferID, oldVal, nil, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "DeleteStockTransferLine", "Stock transfer line deleted", "delete_stock_transfer_line", nil, false, "")
}
