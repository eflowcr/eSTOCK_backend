package controllers

import (
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
