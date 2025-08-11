package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type InventoryController struct {
	Service services.InventoryService
}

func NewInventoryController(service services.InventoryService) *InventoryController {
	return &InventoryController{
		Service: service,
	}
}

func (c *InventoryController) GetAllInventory(ctx *gin.Context) {
	inventory, response := c.Service.GetAllInventory()

	if response != nil {
		tools.Response(ctx, "GetAllInventory", false, response.Message, "get_all_inventory", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllInventory", true, "Inventory retrieved successfully", "get_all_inventory", inventory, false, "")
}

func (c *InventoryController) CreateInventory(ctx *gin.Context) {
	var request requests.CreateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateInventory", false, "Invalid request payload", "create_inventory", nil, false, "")
		return
	}

	response := c.Service.CreateInventory(&request)
	if response != nil {
		tools.Response(ctx, "CreateInventory", false, response.Message, "create_inventory", nil, false, "")
		return
	}

	tools.Response(ctx, "CreateInventory", true, "Inventory created successfully", "create_inventory", nil, false, "")
}

func (c *InventoryController) UpdateInventory(ctx *gin.Context) {
	var request requests.UpdateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "UpdateInventory", false, "Invalid request payload", "update_inventory", nil, false, "")
		return
	}

	response := c.Service.UpdateInventory(&request)
	if response != nil {
		tools.Response(ctx, "UpdateInventory", false, response.Message, "update_inventory", nil, false, "")
		return
	}

	tools.Response(ctx, "UpdateInventory", true, "Inventory updated successfully", "update_inventory", nil, false, "")
}

func (c *InventoryController) DeleteInventory(ctx *gin.Context) {
	id := ctx.Param("id")
	location := ctx.Param("location")

	response := c.Service.DeleteInventory(id, location)
	if response != nil {
		tools.Response(ctx, "DeleteInventory", false, response.Message, "delete_inventory", nil, false, "")
		return
	}

	tools.Response(ctx, "DeleteInventory", true, "Inventory deleted successfully", "delete_inventory", nil, false, "")
}
