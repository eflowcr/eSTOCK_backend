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

func (c *InventoryController) ExportInventoryToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportInventoryToExcel()
	if response != nil {
		tools.Response(ctx, "ExportInventoryToExcel", false, response.Message, "export_inventory_to_excel", nil, false, "")
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="inventory.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *InventoryController) GetInventoryLots(ctx *gin.Context) {
	inventoryID, err := tools.StringToInt(ctx.Param("id"))

	if err != nil {
		tools.Response(ctx, "GetInventoryLots", false, "Invalid inventory ID", "get_inventory_lots", nil, false, "")
		return
	}

	lots, response := c.Service.GetInventoryLots(inventoryID)
	if response != nil {
		tools.Response(ctx, "GetInventoryLots", false, response.Message, "get_inventory_lots", nil, false, "")
		return
	}

	tools.Response(ctx, "GetInventoryLots", true, "Inventory lots retrieved successfully", "get_inventory_lots", lots, false, "")
}

func (c *InventoryController) GetInventorySerials(ctx *gin.Context) {
	inventoryID, err := tools.StringToInt(ctx.Param("id"))

	if err != nil {
		tools.Response(ctx, "GetInventorySerials", false, "Invalid inventory ID", "get_inventory_serials", nil, false, "")
		return
	}

	serials, response := c.Service.GetInventorySerials(inventoryID)
	if response != nil {
		tools.Response(ctx, "GetInventorySerials", false, response.Message, "get_inventory_serials", nil, false, "")
		return
	}

	tools.Response(ctx, "GetInventorySerials", true, "Inventory serials retrieved successfully", "get_inventory_serials", serials, false, "")
}

func (c *InventoryController) CreateInventoryLot(ctx *gin.Context) {
	var request requests.CreateInventoryLotRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateInventoryLot", false, "Invalid request payload", "create_inventory_lot", nil, false, "")
		return
	}

	response := c.Service.CreateInventoryLot(&request)
	if response != nil {
		tools.Response(ctx, "CreateInventoryLot", false, response.Message, "create_inventory_lot", nil, false, "")
		return
	}

	tools.Response(ctx, "CreateInventoryLot", true, "Inventory lot created successfully", "create_inventory_lot", nil, false, "")
}
