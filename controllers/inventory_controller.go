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
