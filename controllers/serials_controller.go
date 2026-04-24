package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// SerialsController is the HTTP entry point for serials master-data.
//
// S3.5 W2-A: tenantID is injected at construction time (from configuration.Config)
// and threaded into every service call so serials are tenant-scoped end-to-end.
type SerialsController struct {
	Service  services.SerialsService
	TenantID string
}

func NewSerialsController(service services.SerialsService, tenantID string) *SerialsController {
	return &SerialsController{Service: service, TenantID: tenantID}
}

func (c *SerialsController) GetSerialByID(ctx *gin.Context) {
	serialID, ok := tools.ParseRequiredParam(ctx, "id", "GetSerialByID", "get_serial_by_id", "ID Serie inválido")
	if !ok {
		return
	}

	serial, resp := c.Service.GetSerialByID(c.TenantID, serialID)
	if resp != nil {
		writeErrorResponse(ctx, "GetSerialByID", "get_serial_by_id", resp)
		return
	}

	if serial == nil {
		tools.ResponseNotFound(ctx, "GetSerialByID", "Serie no encontrada", "get_serial_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetSerialByID", "Serie obtenida con éxito", "get_serial_by_id", serial, false, "")
}

func (c *SerialsController) GetSerialsBySKU(ctx *gin.Context) {
	sku := ctx.Param("sku")
	if sku == "" {
		tools.ResponseBadRequest(ctx, "GetSerials", "Falta el parámetro SKU", "get_serials")
		return
	}

	serials, resp := c.Service.GetSerialsBySKU(c.TenantID, sku)
	if resp != nil {
		writeErrorResponse(ctx, "GetSerials", "get_serials", resp)
		return
	}

	tools.ResponseOK(ctx, "GetSerials", "Series obtenidas con éxito", "get_serials", serials, false, "")
}

func (c *SerialsController) CreateSerial(ctx *gin.Context) {
	var request requests.CreateSerialRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateSerial", "Entrada inválida", "create_serial")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateSerial", "create_serial", errs)
		return
	}

	resp := c.Service.Create(c.TenantID, &request)
	if resp != nil {
		writeErrorResponse(ctx, "CreateSerial", "create_serial", resp)
		return
	}

	tools.ResponseCreated(ctx, "CreateSerial", "Serie creada con éxito", "create_serial", nil, false, "")
}

func (c *SerialsController) UpdateSerial(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateSerial", "update_serial", "ID de serie inválido")
	if !ok {
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateSerial", "Cuerpo de solicitud inválido", "update_serial")
		return
	}

	resp := c.Service.UpdateSerial(c.TenantID, id, data)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateSerial", "update_serial", resp)
		return
	}

	tools.ResponseOK(ctx, "UpdateSerial", "Serie actualizada con éxito", "update_serial", nil, false, "")
}

func (c *SerialsController) DeleteSerial(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteSerial", "delete_serial", "ID de serie inválido")
	if !ok {
		return
	}

	response := c.Service.Delete(c.TenantID, id)
	if response != nil {
		writeErrorResponse(ctx, "DeleteSerial", "delete_serial", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteSerial", "Serie eliminada con éxito", "delete_serial", nil, false, "")
}
