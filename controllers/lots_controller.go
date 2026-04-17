package controllers

import (
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
		writeErrorResponse(ctx, "GetAllLots", "get_all_lots", response)
		return
	}

	if len(lots) == 0 {
		tools.ResponseOK(ctx, "GetAllLots", "No lots found", "get_all_lots", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllLots", "Lots retrieved successfully", "get_all_lots", lots, false, "")
}

func (c *LotsController) GetLotsBySKU(ctx *gin.Context) {
	sku := ctx.Param("sku")
	lots, response := c.Service.GetLotsBySKU(&sku)

	if response != nil {
		writeErrorResponse(ctx, "GetLotsBySKU", "get_lots_by_sku", response)
		return
	}

	if len(lots) == 0 {
		tools.ResponseOK(ctx, "GetLotsBySKU", "No lots found for the given SKU", "get_lots_by_sku", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetLotsBySKU", "Lots retrieved successfully", "get_lots_by_sku", lots, false, "")
}

func (c *LotsController) CreateLot(ctx *gin.Context) {
	var request requests.CreateLotRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateLot", "Invalid request data", "create_lot")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateLot", "create_lot", errs)
		return
	}

	lotResponse := c.Service.Create(&request)
	if lotResponse != nil {
		writeErrorResponse(ctx, "CreateLot", "create_lot", lotResponse)
		return
	}

	tools.ResponseCreated(ctx, "CreateLot", "Lot created successfully", "create_lot", nil, false, "")
}

func (c *LotsController) UpdateLot(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateLot", "update_lot", "Invalid lot ID")
	if !ok {
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateLot", "Invalid request data", "update_lot")
		return
	}

	response := c.Service.UpdateUpdateLot(id, data)
	if response != nil {
		writeErrorResponse(ctx, "UpdateLot", "update_lot", response)
		return
	}

	tools.ResponseOK(ctx, "UpdateLot", "Lot updated successfully", "update_lot", nil, false, "")
}

func (c *LotsController) DeleteLot(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteLot", "delete_lot", "Invalid lot ID")
	if !ok {
		return
	}

	response := c.Service.DeleteLot(id)
	if response != nil {
		writeErrorResponse(ctx, "DeleteLot", "delete_lot", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteLot", "Lot deleted successfully", "delete_lot", nil, false, "")
}

// GetLotTrace handles GET /api/lots/:id/trace
func (c *LotsController) GetLotTrace(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetLotTrace", "get_lot_trace", "Invalid lot ID")
	if !ok {
		return
	}

	trace, resp := c.Service.GetTrace(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetLotTrace", "get_lot_trace", resp)
		return
	}
	tools.ResponseOK(ctx, "GetLotTrace", "Trazabilidad de lote obtenida", "get_lot_trace", trace, false, "")
}
