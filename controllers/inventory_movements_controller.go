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

	movements, response := c.Service.GetAllInventoryMovements()

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

func (c *InventoryMovementsController) GetMovementsBySku(ctx *gin.Context) {
	sku := ctx.Param("sku")

	movements, response := c.Service.GetMovementsBySku(sku)

	if response != nil {
		tools.Response(ctx, "GetMovementsBySku", false, response.Message, "get_movements_by_sku", nil, false, "", response.Handled)
		return
	}

	if len(movements) == 0 {
		tools.Response(ctx, "GetMovementsBySku", true, "No se encontraron movimientos de inventario para el SKU proporcionado", "get_movements_by_sku", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetMovementsBySku", true, "Movimientos de inventario obtenidos con éxito", "get_movements_by_sku", movements, false, "", false)
}
