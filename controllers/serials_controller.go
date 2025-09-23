package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type SerialsController struct {
	Service services.SerialsService
}

func NewSerialsController(service services.SerialsService) *SerialsController {
	return &SerialsController{Service: service}
}

func (c *SerialsController) GetSerialByID(ctx *gin.Context) {
	id := ctx.Param("id")

	// Convert id to uint
	serialID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		tools.Response(ctx, "GetSerialByID", false, "ID Serie inválido", "get_serial_by_id", nil, false, "", false)
		return
	}

	serial, resp := c.Service.GetSerialByID(int(serialID))
	if resp != nil {
		tools.Response(ctx, "GetSerialByID", false, resp.Message, "get_serial_by_id", nil, false, "", resp.Handled)
		return
	}

	if serial == nil {
		tools.Response(ctx, "GetSerialByID", false, "Serie no encontrada", "get_serial_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetSerialByID", true, "Serie obtenida con éxito", "get_serial_by_id", serial, false, "", false)
}

func (c *SerialsController) GetSerialsBySKU(ctx *gin.Context) {
	sku := ctx.Param("sku")
	if sku == "" {
		tools.Response(ctx, "GetSerials", false, "Falta el parámetro SKU", "get_serials", nil, false, "", false)
		return
	}

	serials, resp := c.Service.GetSerialsBySKU(sku)
	if resp != nil {
		tools.Response(ctx, "GetSerials", false, resp.Message, "get_serials", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "GetSerials", true, "Series obtenidas con éxito", "get_serials", serials, false, "", false)
}

func (c *SerialsController) CreateSerial(ctx *gin.Context) {
	var request requests.CreateSerialRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateSerial", false, "Entrada inválida", "create_serial", nil, false, "", false)
		return
	}

	resp := c.Service.Create(&request)
	if resp != nil {
		tools.Response(ctx, "CreateSerial", false, resp.Message, "create_serial", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "CreateSerial", true, "Serie creada con éxito", "create_serial", nil, false, "", false)
}

func (c *SerialsController) UpdateSerial(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "UpdateSerial", false, "ID de serie inválido", "update_serial", nil, false, "", false)
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateSerial", false, "Cuerpo de solicitud inválido", "update_serial", nil, false, "", false)
		return
	}

	resp := c.Service.UpdateSerial(id, data)
	if resp != nil {
		tools.Response(ctx, "UpdateSerial", false, resp.Message, "update_serial", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "UpdateSerial", true, "Serie actualizada con éxito", "update_serial", nil, false, "", false)
}

func (c *SerialsController) DeleteSerial(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "DeleteSerial", false, "ID de serie inválido", "delete_serial", nil, false, "", false)
		return
	}

	response := c.Service.Delete(id)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "DeleteSerial", false, response.Message, "delete_serial", nil, false, "", true)
		} else {
			tools.Response(ctx, "DeleteSerial", false, "Ocurrió un error interno", "delete_serial", nil, false, "", false)
		}
		return
	}

	tools.Response(ctx, "DeleteSerial", true, "Serie eliminada con éxito", "delete_serial", nil, false, "", false)
}
