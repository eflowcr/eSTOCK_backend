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
		tools.Response(ctx, "GetAllInventoryMovements", false, response.Message, "get_all_inventory_movements", nil, false, "", response.Handled)
		return
	}

	if len(movements) == 0 {
		tools.Response(ctx, "GetAllInventoryMovements", true, "No se encontraron movimientos de inventario", "get_all_inventory_movements", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetAllInventoryMovements", true, "Movimientos de inventario obtenidos con éxito", "get_all_inventory_movements", movements, false, "", false)
}
