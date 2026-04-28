package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type InventoryCountsController struct {
	Service   services.InventoryCountsService
	JWTSecret string
}

func NewInventoryCountsController(service services.InventoryCountsService, jwtSecret string) *InventoryCountsController {
	return &InventoryCountsController{Service: service, JWTSecret: jwtSecret}
}

func (c *InventoryCountsController) List(ctx *gin.Context) {
	status := ctx.Query("status")
	locationID := ctx.Query("location_id")
	list, resp := c.Service.List(status, locationID)
	if resp != nil {
		writeErrorResponse(ctx, "ListInventoryCounts", "list_inventory_counts", resp)
		return
	}
	tools.ResponseOK(ctx, "ListInventoryCounts", "Conteos obtenidos", "list_inventory_counts", list, false, "")
}

func (c *InventoryCountsController) GetDetail(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetInventoryCount", "get_inventory_count", "ID de conteo inválido")
	if !ok {
		return
	}
	detail, resp := c.Service.GetDetail(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetInventoryCount", "get_inventory_count", resp)
		return
	}
	tools.ResponseOK(ctx, "GetInventoryCount", "Conteo obtenido", "get_inventory_count", detail, false, "")
}

func (c *InventoryCountsController) Create(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "CreateInventoryCount", "Token inválido", "invalid_token")
		return
	}
	var body requests.CreateInventoryCount
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateInventoryCount", "Cuerpo de solicitud inválido", "create_inventory_count")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateInventoryCount", "create_inventory_count", errs)
		return
	}
	created, resp := c.Service.Create(userID, &body)
	if resp != nil {
		writeErrorResponse(ctx, "CreateInventoryCount", "create_inventory_count", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateInventoryCount", "Conteo creado", "create_inventory_count", created, false, "")
}

func (c *InventoryCountsController) Start(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "StartInventoryCount", "start_inventory_count", "ID de conteo inválido")
	if !ok {
		return
	}
	if resp := c.Service.Start(id); resp != nil {
		writeErrorResponse(ctx, "StartInventoryCount", "start_inventory_count", resp)
		return
	}
	tools.ResponseOK(ctx, "StartInventoryCount", "Conteo iniciado", "start_inventory_count", nil, false, "")
}

func (c *InventoryCountsController) Cancel(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CancelInventoryCount", "cancel_inventory_count", "ID de conteo inválido")
	if !ok {
		return
	}
	if resp := c.Service.Cancel(id); resp != nil {
		writeErrorResponse(ctx, "CancelInventoryCount", "cancel_inventory_count", resp)
		return
	}
	tools.ResponseOK(ctx, "CancelInventoryCount", "Conteo cancelado", "cancel_inventory_count", nil, false, "")
}

func (c *InventoryCountsController) ScanLine(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "ScanCountLine", "scan_count_line", "ID de conteo inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "ScanCountLine", "Token inválido", "invalid_token")
		return
	}
	var body requests.ScanCountLine
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "ScanCountLine", "Cuerpo de solicitud inválido", "scan_count_line")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "ScanCountLine", "scan_count_line", errs)
		return
	}
	line, resp := c.Service.ScanLine(id, userID, &body)
	if resp != nil {
		writeErrorResponse(ctx, "ScanCountLine", "scan_count_line", resp)
		return
	}
	tools.ResponseOK(ctx, "ScanCountLine", "Línea registrada", "scan_count_line", line, false, "")
}

func (c *InventoryCountsController) Submit(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "SubmitInventoryCount", "submit_inventory_count", "ID de conteo inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "SubmitInventoryCount", "Token inválido", "invalid_token")
		return
	}
	updated, resp := c.Service.Submit(id, userID)
	if resp != nil {
		writeErrorResponse(ctx, "SubmitInventoryCount", "submit_inventory_count", resp)
		return
	}
	tools.ResponseOK(ctx, "SubmitInventoryCount", "Conteo enviado", "submit_inventory_count", updated, false, "")
}
