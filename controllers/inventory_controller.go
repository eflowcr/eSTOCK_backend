package controllers

import (
	"io"

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

func (c *InventoryController) Trend(ctx *gin.Context) {
	sku := ctx.Param("sku")

	trend, response := c.Service.Trend(sku)
	if response != nil {
		tools.Response(ctx, "Trend", false, response.Message, "inventory_trend", nil, false, "")
		return
	}

	tools.Response(ctx, "Trend", true, "Inventory trend retrieved successfully", "inventory_trend", trend, false, "")
}

func (c *InventoryController) ImportInventoryFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "File upload error", "import_inventory_from_excel", nil, false, "")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "Failed to open file: "+err.Error(), "import_inventory_from_excel", nil, false, "")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "Failed to read file content: "+err.Error(), "import_inventory_from_excel", nil, false, "")
		return
	}

	imported, errorResponses := c.Service.ImportInventoryFromExcel(fileBytes)

	if len(imported) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		tools.Response(ctx, "ImportInventoryFromExcel", false, resp.Message, "import_inventory_from_excel", nil, false, "")
		return
	}

	tools.Response(ctx, "ImportInventoryFromExcel", true, "Inventory imported successfully", "import_inventory_from_excel", gin.H{
		"imported_items": imported,
		"errors":         errorResponses,
	}, false, "")
}
