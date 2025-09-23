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
		tools.Response(ctx, "GetAllInventory", false, response.Message, "get_all_inventory", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "GetAllInventory", true, "Inventory retrieved successfully", "get_all_inventory", inventory, false, "", false)
}

func (c *InventoryController) CreateInventory(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	var request requests.CreateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateInventory", false, "Invalid request payload", "create_inventory", nil, false, "", false)
		return
	}

	response := c.Service.CreateInventory(userId, &request)
	if response != nil {
		tools.Response(ctx, "CreateInventory", false, response.Message, "create_inventory", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateInventory", true, "Inventory created successfully", "create_inventory", nil, false, "", false)
}

func (c *InventoryController) UpdateInventory(ctx *gin.Context) {
	var request requests.UpdateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "UpdateInventory", false, "Invalid request payload", "update_inventory", nil, false, "", false)
		return
	}

	response := c.Service.UpdateInventory(&request)
	if response != nil {
		tools.Response(ctx, "UpdateInventory", false, response.Message, "update_inventory", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "UpdateInventory", true, "Inventory updated successfully", "update_inventory", nil, false, "", false)
}

func (c *InventoryController) DeleteInventory(ctx *gin.Context) {
	id := ctx.Param("id")
	location := ctx.Param("location")

	response := c.Service.DeleteInventory(id, location)
	if response != nil {
		tools.Response(ctx, "DeleteInventory", false, response.Message, "delete_inventory", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeleteInventory", true, "Inventory deleted successfully", "delete_inventory", nil, false, "", false)
}

func (c *InventoryController) Trend(ctx *gin.Context) {
	sku := ctx.Param("sku")

	trend, response := c.Service.Trend(sku)
	if response != nil {
		tools.Response(ctx, "Trend", false, response.Message, "inventory_trend", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "Trend", true, "Inventory trend retrieved successfully", "inventory_trend", trend, false, "", false)
}

func (c *InventoryController) ImportInventoryFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "File upload error", "import_inventory_from_excel", nil, false, "", false)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "Failed to open file: "+err.Error(), "import_inventory_from_excel", nil, false, "", false)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportInventoryFromExcel", false, "Failed to read file content: "+err.Error(), "import_inventory_from_excel", nil, false, "", false)
		return
	}

	imported, errorResponses := c.Service.ImportInventoryFromExcel(userId, fileBytes)

	if len(imported) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		tools.Response(ctx, "ImportInventoryFromExcel", false, resp.Message, "import_inventory_from_excel", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "ImportInventoryFromExcel", true, "Inventory imported successfully", "import_inventory_from_excel", gin.H{
		"imported_items": imported,
		"errors":         errorResponses,
	}, false, "", false)
}

func (c *InventoryController) ExportInventoryToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportInventoryToExcel()
	if response != nil {
		tools.Response(ctx, "ExportInventoryToExcel", false, response.Message, "export_inventory_to_excel", nil, false, "", response.Handled)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="inventory.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *InventoryController) GetInventoryLots(ctx *gin.Context) {
	inventoryID, err := tools.StringToInt(ctx.Param("id"))

	if err != nil {
		tools.Response(ctx, "GetInventoryLots", false, "Invalid inventory ID", "get_inventory_lots", nil, false, "", false)
		return
	}

	lots, response := c.Service.GetInventoryLots(inventoryID)
	if response != nil {
		tools.Response(ctx, "GetInventoryLots", false, response.Message, "get_inventory_lots", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "GetInventoryLots", true, "Inventory lots retrieved successfully", "get_inventory_lots", lots, false, "", false)
}

func (c *InventoryController) GetInventorySerials(ctx *gin.Context) {
	inventoryID, err := tools.StringToInt(ctx.Param("id"))

	if err != nil {
		tools.Response(ctx, "GetInventorySerials", false, "Invalid inventory ID", "get_inventory_serials", nil, false, "", false)
		return
	}

	serials, response := c.Service.GetInventorySerials(inventoryID)
	if response != nil {
		tools.Response(ctx, "GetInventorySerials", false, response.Message, "get_inventory_serials", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "GetInventorySerials", true, "Inventory serials retrieved successfully", "get_inventory_serials", serials, false, "", false)
}

func (c *InventoryController) CreateInventoryLot(ctx *gin.Context) {
	id, _ := tools.StringToInt(ctx.Param("id"))

	var request requests.CreateInventoryLotRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateInventoryLot", false, "Invalid request payload", "create_inventory_lot", nil, false, "", false)
		return
	}

	response := c.Service.CreateInventoryLot(id, &request)
	if response != nil {
		tools.Response(ctx, "CreateInventoryLot", false, response.Message, "create_inventory_lot", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateInventoryLot", true, "Inventory lot created successfully", "create_inventory_lot", nil, false, "", false)
}

func (c *InventoryController) DeleteInventoryLot(ctx *gin.Context) {
	id, err := tools.StringToInt(ctx.Param("id"))
	if err != nil {
		tools.Response(ctx, "DeleteInventoryLot", false, "Invalid lot ID", "delete_inventory_lot", nil, false, "", false)
		return
	}

	response := c.Service.DeleteInventoryLot(id)
	if response != nil {
		tools.Response(ctx, "DeleteInventoryLot", false, response.Message, "delete_inventory_lot", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeleteInventoryLot", true, "Inventory lot deleted successfully", "delete_inventory_lot", nil, false, "", false)
}

func (c *InventoryController) CreateInventorySerial(ctx *gin.Context) {
	id, _ := tools.StringToInt(ctx.Param("id"))

	var request requests.CreateInventorySerial
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateInventorySerial", false, "Invalid request payload", "create_inventory_serial", nil, false, "", false)
		return
	}

	response := c.Service.CreateInventorySerial(id, &request)
	if response != nil {
		tools.Response(ctx, "CreateInventorySerial", false, response.Message, "create_inventory_serial", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateInventorySerial", true, "Inventory serial created successfully", "create_inventory_serial", nil, false, "", false)
}

func (c *InventoryController) DeleteInventorySerial(ctx *gin.Context) {
	id, err := tools.StringToInt(ctx.Param("id"))
	if err != nil {
		tools.Response(ctx, "DeleteInventorySerial", false, "Invalid serial ID", "delete_inventory_serial", nil, false, "", false)
		return
	}

	response := c.Service.DeleteInventorySerial(id)
	if response != nil {
		tools.Response(ctx, "DeleteInventorySerial", false, response.Message, "delete_inventory_serial", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeleteInventorySerial", true, "Inventory serial deleted successfully", "delete_inventory_serial", nil, false, "", false)
}
