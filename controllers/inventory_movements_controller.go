package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type InventoryMovementsController struct {
	Service services.InventoryMovementsService
}

func NewInventoryMovementsController(service services.InventoryMovementsService) *InventoryMovementsController {
	return &InventoryMovementsController{
		Service: service,
	}
}

func (c *InventoryMovementsController) GetAllInventoryMovements(ctx *gin.Context) {
	sku := ctx.Param("sku")

	movements, response := c.Service.GetAllInventoryMovements(sku)

	if response != nil {
		writeErrorResponse(ctx, "GetAllInventoryMovements", "get_all_inventory_movements", response)
		return
	}

	if len(movements) == 0 {
		tools.ResponseOK(ctx, "GetAllInventoryMovements", "No se encontraron movimientos de inventario", "get_all_inventory_movements", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllInventoryMovements", "Movimientos de inventario obtenidos con éxito", "get_all_inventory_movements", movements, false, "")
}

func (c *InventoryMovementsController) ListMovements(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "500"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	f := ports.MovementsFilter{
		SKU:           ctx.Query("sku"),
		Location:      ctx.Query("location"),
		LotID:         ctx.Query("lot_id"),
		MovementType:  ctx.Query("movement_type"),
		ReferenceType: ctx.Query("reference_type"),
		UserID:        ctx.Query("user_id"),
		From:          ctx.Query("from"),
		To:            ctx.Query("to"),
		Limit:         limit,
		Offset:        offset,
	}

	movements, response := c.Service.ListMovements(f)
	if response != nil {
		writeErrorResponse(ctx, "ListMovements", "list_movements", response)
		return
	}

	tools.ResponseOK(ctx, "ListMovements", "Movimientos de inventario obtenidos", "list_movements", movements, false, "")
}
