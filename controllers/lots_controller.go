package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type LotsController struct {
	Service services.LotsService
}

func NewLotsController(service services.LotsService) *LotsController {
	return &LotsController{
		Service: service,
	}
}

func (c *LotsController) GetAllLots(ctx *gin.Context) {
	lots, response := c.Service.GetAllLots()

	if response != nil {
		tools.Response(ctx, "GetAllLots", false, response.Message, "get_all_lots", nil, false, "")
		return
	}

	if len(lots) == 0 {
		tools.Response(ctx, "GetAllLots", true, "No lots found", "get_all_lots", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllLots", true, "Lots retrieved successfully", "get_all_lots", lots, false, "")
}

func (c *LotsController) GetLotsBySKU(ctx *gin.Context) {
	sku := ctx.Param("sku")
	lots, response := c.Service.GetLotsBySKU(&sku)

	if response != nil {
		tools.Response(ctx, "GetLotsBySKU", false, response.Message, "get_lots_by_sku", nil, false, "")
		return
	}

	if len(lots) == 0 {
		tools.Response(ctx, "GetLotsBySKU", true, "No lots found for the given SKU", "get_lots_by_sku", nil, false, "")
		return
	}

	tools.Response(ctx, "GetLotsBySKU", true, "Lots retrieved successfully", "get_lots_by_sku", lots, false, "")
}

func (c *LotsController) CreateLot(ctx *gin.Context) {
	var request requests.CreateLotRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateLot", false, "Invalid request data", "create_lot", nil, false, "")
		return
	}

	lotResponse := c.Service.Create(&request)
	if lotResponse != nil {
		tools.Response(ctx, "CreateLot", false, lotResponse.Message, "create_lot", nil, false, "")
		return
	}

	tools.Response(ctx, "CreateLot", true, "Lot created successfully", "create_lot", nil, false, "")
}

func (c *LotsController) UpdateLot(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "UpdateLot", false, "Invalid lot ID", "update_lot", nil, false, "")
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateLot", false, "Invalid request data", "update_lot", nil, false, "")
		return
	}

	response := c.Service.UpdateUpdateLot(id, data)
	if response != nil {
		tools.Response(ctx, "UpdateLot", false, response.Message, "update_lot", nil, false, "")
		return
	}

	tools.Response(ctx, "UpdateLot", true, "Lot updated successfully", "update_lot", nil, false, "")
}

func (c *LotsController) DeleteLot(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "DeleteLot", false, "Invalid lot ID", "delete_lot", nil, false, "")
		return
	}

	response := c.Service.DeleteLot(id)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "DeleteLot", false, response.Message, "delete_lot", nil, false, "")
		} else {
			tools.Response(ctx, "DeleteLot", false, "Internal error occurred", "delete_lot", nil, false, "")
		}
		return
	}

	tools.Response(ctx, "DeleteLot", true, "Lot deleted successfully", "delete_lot", nil, false, "")
}
