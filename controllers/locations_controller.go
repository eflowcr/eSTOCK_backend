package controllers

import (
	"io"
	"strconv"

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
		tools.Response(ctx, "GetAllLocations", false, response.Message, "get_all_locations", nil, false, "", response.Handled)
		return
	}

	if len(locations) == 0 {
		tools.Response(ctx, "GetAllLocations", true, "No se encontraron ubicaciones", "get_all_locations", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetAllLocations", true, "Ubicaciones obtenidas con éxito", "get_all_locations", locations, false, "", false)
}

func (c *LocationsController) GetLocationByID(ctx *gin.Context) {
	id := ctx.Param("id")
	location, response := c.Service.GetLocationByID(id)

	if response != nil {
		tools.Response(ctx, "GetLocationByID", false, response.Message, "get_location_by_id", nil, false, "", response.Handled)
		return
	}

	if location == nil {
		tools.Response(ctx, "GetLocationByID", true, "Ubicación no encontrada", "get_location_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetLocationByID", true, "Ubicación obtenida con éxito", "get_location_by_id", location, false, "", false)
}

func (c *LocationsController) CreateLocation(ctx *gin.Context) {
	var body requests.Location

	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.Response(ctx, "CreateLocation", false, "Cuerpo de solicitud inválido", "create_location", nil, false, "", false)
		return
	}

	resp := c.Service.CreateLocation(&body)

	if resp != nil {
		tools.Response(ctx, "CreateLocation", false, resp.Message, "create_location", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "CreateLocation", true, "Ubicación creada con éxito", "create_location", nil, false, "", false)
}

func (c *LocationsController) UpdateLocation(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "UpdateLocation", false, "ID de ubicación inválido", "update_location", nil, false, "", false)
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateLocation", false, "Cuerpo de solicitud inválido", "update_location", nil, false, "", false)
		return
	}

	response := c.Service.UpdateLocation(id, data)
	if response != nil {
		tools.Response(ctx, "UpdateLocation", false, response.Message, "update_location", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "UpdateLocation", true, "Ubicación actualizada con éxito", "update_location", nil, false, "", false)
}

func (c *LocationsController) DeleteLocation(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "DeleteLocation", false, "ID de ubicación inválido", "delete_location", nil, false, "", false)
		return
	}

	response := c.Service.DeleteLocation(id)
	if response != nil {
		tools.Response(ctx, "DeleteLocation", false, response.Message, "delete_location", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeleteLocation", true, "Ubicación eliminada con éxito", "delete_location", nil, false, "", false)
}

func (c *LocationsController) ImportLocationsFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al subir el archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al abrir el archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al leer el contenido del archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}

	importedLocations, errorResponses := c.Service.ImportLocationsFromExcel(fileBytes)

	if len(importedLocations) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		tools.Response(ctx, "ImportLocationsFromExcel", false, resp.Message, "import_locations_from_excel", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "ImportLocationsFromExcel", true, "Ubicaciones importadas con éxito", "import_locations_from_excel", gin.H{
		"imported_locations": importedLocations,
		"errors":             errorResponses,
	}, false, "", false)
}

func (c *LocationsController) ExportLocationsToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportLocationsToExcel()
	if response != nil {
		tools.Response(ctx, "ExportLocationsToExcel", false, response.Message, "export_locations_to_excel", nil, false, "", response.Handled)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="locations.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}
