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
		tools.Response(ctx, "GetSerialByID", false, "Invalid serial ID", "get_serial_by_id", nil, false, "")
		return
	}

	serial, resp := c.Service.GetSerialByID(int(serialID))
	if resp != nil {
		tools.Response(ctx, "GetSerialByID", false, resp.Message, "get_serial_by_id", nil, false, "")
		return
	}

	if serial == nil {
		tools.Response(ctx, "GetSerialByID", false, "Serial not found", "get_serial_by_id", nil, false, "")
		return
	}

	tools.Response(ctx, "GetSerialByID", true, "Serial retrieved successfully", "get_serial_by_id", serial, false, "")
}

func (c *SerialsController) GetSerialsBySKU(ctx *gin.Context) {
	sku := ctx.Param("sku")
	if sku == "" {
		tools.Response(ctx, "GetSerials", false, "Missing SKU parameter", "get_serials", nil, false, "")
		return
	}

	serials, resp := c.Service.GetSerialsBySKU(sku)
	if resp != nil {
		tools.Response(ctx, "GetSerials", false, resp.Message, "get_serials", nil, false, "")
		return
	}

	tools.Response(ctx, "GetSerials", true, "Serials retrieved successfully", "get_serials", serials, false, "")
}

func (c *SerialsController) CreateSerial(ctx *gin.Context) {
	var request requests.CreateSerialRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateSerial", false, "Invalid input", "create_serial", nil, false, "")
		return
	}

	resp := c.Service.Create(&request)
	if resp != nil {
		tools.Response(ctx, "CreateSerial", false, resp.Message, "create_serial", nil, false, "")
		return
	}

	tools.Response(ctx, "CreateSerial", true, "Serial created successfully", "create_serial", nil, false, "")
}

func (c *SerialsController) UpdateSerial(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "UpdateSerial", false, "Invalid serial ID", "update_serial", nil, false, "")
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateSerial", false, "Invalid request data", "update_serial", nil, false, "")
		return
	}

	resp := c.Service.UpdateSerial(id, data)
	if resp != nil {
		tools.Response(ctx, "UpdateSerial", false, resp.Message, "update_serial", nil, false, "")
		return
	}

	tools.Response(ctx, "UpdateSerial", true, "Serial updated successfully", "update_serial", nil, false, "")
}

func (c *SerialsController) DeleteSerial(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "DeleteSerial", false, "Invalid serial ID", "delete_serial", nil, false, "")
		return
	}

	response := c.Service.Delete(id)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "DeleteSerial", false, response.Message, "delete_serial", nil, false, "")
		} else {
			tools.Response(ctx, "DeleteSerial", false, "Internal error occurred", "delete_serial", nil, false, "")
		}
		return
	}

	tools.Response(ctx, "DeleteSerial", true, "Serial deleted successfully", "delete_serial", nil, false, "")
}
