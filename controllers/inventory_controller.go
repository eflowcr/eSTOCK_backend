package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type InventoryController struct {
	Service   services.InventoryService
	JWTSecret string
}

func NewInventoryController(service services.InventoryService, jwtSecret string) *InventoryController {
	return &InventoryController{
		Service:   service,
		JWTSecret: jwtSecret,
	}
}

func (c *InventoryController) GetAllInventory(ctx *gin.Context) {
	inventory, response := c.Service.GetAllInventory()

	if response != nil {
		writeErrorResponse(ctx, "GetAllInventory", "get_all_inventory", response)
		return
	}

	tools.ResponseOK(ctx, "GetAllInventory", "Inventario obtenido con éxito", "get_all_inventory", inventory, false, "")
}

func (c *InventoryController) GetInventoryBySkuAndLocation(ctx *gin.Context) {
	sku := ctx.Param("sku")
	location := ctx.Param("location")
	if sku == "" || location == "" {
		tools.ResponseBadRequest(ctx, "GetInventoryBySkuAndLocation", "SKU y ubicación son requeridos", "get_inventory_by_sku_location")
		return
	}

	item, response := c.Service.GetInventoryBySkuAndLocation(sku, location)
	if response != nil {
		writeErrorResponse(ctx, "GetInventoryBySkuAndLocation", "get_inventory_by_sku_location", response)
		return
	}
	if item == nil {
		tools.ResponseNotFound(ctx, "GetInventoryBySkuAndLocation", "No existe inventario para este SKU en la ubicación indicada", "get_inventory_by_sku_location")
		return
	}

	tools.ResponseOK(ctx, "GetInventoryBySkuAndLocation", "Inventario obtenido con éxito", "get_inventory_by_sku_location", item, false, "")
}

func (c *InventoryController) GetPickSuggestions(ctx *gin.Context) {
	sku := ctx.Param("sku")
	if sku == "" {
		tools.ResponseBadRequest(ctx, "GetPickSuggestions", "SKU es requerido", "get_pick_suggestions")
		return
	}
	list, response := c.Service.GetPickSuggestionsBySKU(sku)
	if response != nil {
		writeErrorResponse(ctx, "GetPickSuggestions", "get_pick_suggestions", response)
		return
	}
	if list == nil {
		list = []dto.PickSuggestion{}
	}
	tools.ResponseOK(ctx, "GetPickSuggestions", "Sugerencias de picking obtenidas", "get_pick_suggestions", list, false, "")
}

func (c *InventoryController) CreateInventory(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	var request requests.CreateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateInventory", "Carga útil de solicitud no válida", "create_inventory")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateInventory", "create_inventory", errs)
		return
	}

	response := c.Service.CreateInventory(userId, &request)
	if response != nil {
		writeErrorResponse(ctx, "CreateInventory", "create_inventory", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateInventory", "Inventario creado con éxito", "create_inventory", nil, false, "")
}

func (c *InventoryController) UpdateInventory(ctx *gin.Context) {
	var request requests.UpdateInventory
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateInventory", "Invalid request payload", "update_inventory")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateInventory", "update_inventory", errs)
		return
	}

	response := c.Service.UpdateInventory(&request)
	if response != nil {
		writeErrorResponse(ctx, "UpdateInventory", "update_inventory", response)
		return
	}

	tools.ResponseOK(ctx, "UpdateInventory", "Inventario actualizado con éxito", "update_inventory", nil, false, "")
}

func (c *InventoryController) DeleteInventory(ctx *gin.Context) {
	id := ctx.Param("id")
	location := ctx.Param("location")

	response := c.Service.DeleteInventory(id, location)
	if response != nil {
		writeErrorResponse(ctx, "DeleteInventory", "delete_inventory", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteInventory", "Inventario eliminado con éxito", "delete_inventory", nil, false, "")
}

func (c *InventoryController) Trend(ctx *gin.Context) {
	sku := ctx.Param("sku")

	trend, response := c.Service.Trend(sku)
	if response != nil {
		writeErrorResponse(ctx, "Trend", "inventory_trend", response)
		return
	}

	tools.ResponseOK(ctx, "Trend", "Tendencia de inventario obtenida con éxito", "inventory_trend", trend, false, "")
}

func (c *InventoryController) ImportInventoryFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportInventoryFromExcel", "Error de carga de archivo", "import_inventory_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportInventoryFromExcel", "Error al abrir el archivo", "import_inventory_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportInventoryFromExcel", "Error al leer el contenido del archivo", "import_inventory_from_excel")
		return
	}

	imported, errorResponses := c.Service.ImportInventoryFromExcel(userId, fileBytes)

	if len(imported) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		writeErrorResponse(ctx, "ImportInventoryFromExcel", "import_inventory_from_excel", resp)
		return
	}

	tools.ResponseOK(ctx, "ImportInventoryFromExcel", "Inventario importado con éxito", "import_inventory_from_excel", gin.H{
		"imported_items": imported,
		"errors":         errorResponses,
	}, false, "")
}

func (c *InventoryController) ExportInventoryToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportInventoryToExcel()
	if response != nil {
		writeErrorResponse(ctx, "ExportInventoryToExcel", "export_inventory_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="inventory.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *InventoryController) GetInventoryLots(ctx *gin.Context) {
	inventoryID, ok := tools.ParseRequiredParam(ctx, "id", "GetInventoryLots", "get_inventory_lots", "Invalid inventory ID")
	if !ok {
		return
	}

	lots, response := c.Service.GetInventoryLots(inventoryID)
	if response != nil {
		writeErrorResponse(ctx, "GetInventoryLots", "get_inventory_lots", response)
		return
	}

	tools.ResponseOK(ctx, "GetInventoryLots", "Lotes de inventario recuperados con éxito", "get_inventory_lots", lots, false, "")
}

func (c *InventoryController) GetInventorySerials(ctx *gin.Context) {
	inventoryID, ok := tools.ParseRequiredParam(ctx, "id", "GetInventorySerials", "get_inventory_serials", "ID de inventario no válido")
	if !ok {
		return
	}

	serials, response := c.Service.GetInventorySerials(inventoryID)
	if response != nil {
		writeErrorResponse(ctx, "GetInventorySerials", "get_inventory_serials", response)
		return
	}

	tools.ResponseOK(ctx, "GetInventorySerials", "Seriales de inventario obtenidos con éxito", "get_inventory_serials", serials, false, "")
}

func (c *InventoryController) CreateInventoryLot(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CreateInventoryLot", "create_inventory_lot", "ID de inventario no válido")
	if !ok {
		return
	}

	var request requests.CreateInventoryLotRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateInventoryLot", "Carga útil de solicitud no válida", "create_inventory_lot")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateInventoryLot", "create_inventory_lot", errs)
		return
	}

	response := c.Service.CreateInventoryLot(id, &request)
	if response != nil {
		writeErrorResponse(ctx, "CreateInventoryLot", "create_inventory_lot", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateInventoryLot", "Lote de inventario creado con éxito", "create_inventory_lot", nil, false, "")
}

func (c *InventoryController) DeleteInventoryLot(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteInventoryLot", "delete_inventory_lot", "ID de lote no válido")
	if !ok {
		return
	}

	response := c.Service.DeleteInventoryLot(id)
	if response != nil {
		writeErrorResponse(ctx, "DeleteInventoryLot", "delete_inventory_lot", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteInventoryLot", "Lote de inventario eliminado con éxito", "delete_inventory_lot", nil, false, "")
}

func (c *InventoryController) CreateInventorySerial(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CreateInventorySerial", "create_inventory_serial", "ID de inventario no válido")
	if !ok {
		return
	}

	var request requests.CreateInventorySerial
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateInventorySerial", "Carga útil de solicitud no válida", "create_inventory_serial")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateInventorySerial", "create_inventory_serial", errs)
		return
	}

	response := c.Service.CreateInventorySerial(id, &request)
	if response != nil {
		writeErrorResponse(ctx, "CreateInventorySerial", "create_inventory_serial", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateInventorySerial", "Serial de inventario creado con éxito", "create_inventory_serial", nil, false, "")
}

func (c *InventoryController) DeleteInventorySerial(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteInventorySerial", "delete_inventory_serial", "ID de serie no válido")
	if !ok {
		return
	}

	response := c.Service.DeleteInventorySerial(id)
	if response != nil {
		writeErrorResponse(ctx, "DeleteInventorySerial", "delete_inventory_serial", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteInventorySerial", "Serial de inventario eliminado con éxito", "delete_inventory_serial", nil, false, "")
}
