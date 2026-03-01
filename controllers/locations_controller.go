package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type LocationsController struct {
	Service services.LocationsService
}

func NewLocationsController(service services.LocationsService) *LocationsController {
	return &LocationsController{
		Service: service,
	}
}

func (c *LocationsController) GetAllLocations(ctx *gin.Context) {
	locations, response := c.Service.GetAllLocations()

	if response != nil {
		writeErrorResponse(ctx, "GetAllLocations", "get_all_locations", response)
		return
	}

	if len(locations) == 0 {
		tools.ResponseOK(ctx, "GetAllLocations", "No se encontraron ubicaciones", "get_all_locations", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllLocations", "Ubicaciones obtenidas con éxito", "get_all_locations", locations, false, "")
}

func (c *LocationsController) GetLocationByID(ctx *gin.Context) {
	id := ctx.Param("id")
	location, response := c.Service.GetLocationByID(id)

	if response != nil {
		writeErrorResponse(ctx, "GetLocationByID", "get_location_by_id", response)
		return
	}

	if location == nil {
		tools.ResponseNotFound(ctx, "GetLocationByID", "Ubicación no encontrada", "get_location_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetLocationByID", "Ubicación obtenida con éxito", "get_location_by_id", location, false, "")
}

func (c *LocationsController) CreateLocation(ctx *gin.Context) {
	var body requests.Location

	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateLocation", "Cuerpo de solicitud inválido", "create_location")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateLocation", "create_location", errs)
		return
	}

	resp := c.Service.CreateLocation(&body)

	if resp != nil {
		writeErrorResponse(ctx, "CreateLocation", "create_location", resp)
		return
	}

	tools.ResponseCreated(ctx, "CreateLocation", "Ubicación creada con éxito", "create_location", nil, false, "")
}

func (c *LocationsController) UpdateLocation(ctx *gin.Context) {
	id, ok := tools.ParseIntParam(ctx, "id", "UpdateLocation", "update_location", "ID de ubicación inválido")
	if !ok {
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateLocation", "Cuerpo de solicitud inválido", "update_location")
		return
	}

	response := c.Service.UpdateLocation(id, data)
	if response != nil {
		writeErrorResponse(ctx, "UpdateLocation", "update_location", response)
		return
	}

	tools.ResponseOK(ctx, "UpdateLocation", "Ubicación actualizada con éxito", "update_location", nil, false, "")
}

func (c *LocationsController) DeleteLocation(ctx *gin.Context) {
	id, ok := tools.ParseIntParam(ctx, "id", "DeleteLocation", "delete_location", "ID de ubicación inválido")
	if !ok {
		return
	}

	response := c.Service.DeleteLocation(id)
	if response != nil {
		writeErrorResponse(ctx, "DeleteLocation", "delete_location", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteLocation", "Ubicación eliminada con éxito", "delete_location", nil, false, "")
}

func (c *LocationsController) ImportLocationsFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportLocationsFromExcel", "Error al subir el archivo", "import_locations_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportLocationsFromExcel", "Error al abrir el archivo", "import_locations_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportLocationsFromExcel", "Error al leer el contenido del archivo", "import_locations_from_excel")
		return
	}

	importedLocations, errorResponses := c.Service.ImportLocationsFromExcel(fileBytes)

	if len(importedLocations) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		writeErrorResponse(ctx, "ImportLocationsFromExcel", "import_locations_from_excel", resp)
		return
	}

	tools.ResponseOK(ctx, "ImportLocationsFromExcel", "Ubicaciones importadas con éxito", "import_locations_from_excel", gin.H{
		"imported_locations": importedLocations,
		"errors":             errorResponses,
	}, false, "")
}

func (c *LocationsController) ExportLocationsToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportLocationsToExcel()
	if response != nil {
		writeErrorResponse(ctx, "ExportLocationsToExcel", "export_locations_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="locations.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}
