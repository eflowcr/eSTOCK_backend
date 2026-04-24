package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// LocationsController is the HTTP entry point for location master-data.
//
// S3.5 W2-A: tenantID is injected at construction time (from configuration.Config)
// and threaded into every service call so locations are tenant-scoped end-to-end.
type LocationsController struct {
	Service  services.LocationsService
	TenantID string
}

func NewLocationsController(service services.LocationsService, tenantID string) *LocationsController {
	return &LocationsController{
		Service:  service,
		TenantID: tenantID,
	}
}

func (c *LocationsController) GetAllLocations(ctx *gin.Context) {
	locations, response := c.Service.GetAllLocations(c.resolveTenantID(ctx))

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
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetLocationByID", "get_location_by_id", "ID de ubicación inválido")
	if !ok {
		return
	}
	location, response := c.Service.GetLocationByID(c.resolveTenantID(ctx), id)

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

	resp := c.Service.CreateLocation(c.resolveTenantID(ctx), &body)

	if resp != nil {
		writeErrorResponse(ctx, "CreateLocation", "create_location", resp)
		return
	}

	tools.ResponseCreated(ctx, "CreateLocation", "Ubicación creada con éxito", "create_location", nil, false, "")
}

func (c *LocationsController) UpdateLocation(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateLocation", "update_location", "ID de ubicación inválido")
	if !ok {
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateLocation", "Cuerpo de solicitud inválido", "update_location")
		return
	}

	response := c.Service.UpdateLocation(c.resolveTenantID(ctx), id, data)
	if response != nil {
		writeErrorResponse(ctx, "UpdateLocation", "update_location", response)
		return
	}

	tools.ResponseOK(ctx, "UpdateLocation", "Ubicación actualizada con éxito", "update_location", nil, false, "")
}

func (c *LocationsController) DeleteLocation(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteLocation", "delete_location", "ID de ubicación inválido")
	if !ok {
		return
	}

	response := c.Service.DeleteLocation(c.resolveTenantID(ctx), id)
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

	imported, skipped, errResp := c.Service.ImportLocationsFromExcel(c.resolveTenantID(ctx), fileBytes)
	if errResp != nil && len(imported) == 0 {
		writeErrorResponse(ctx, "ImportLocationsFromExcel", "import_locations_from_excel", errResp)
		return
	}

	tools.ResponseOK(ctx, "ImportLocationsFromExcel", "Ubicaciones importadas con éxito", "import_locations_from_excel", gin.H{
		"successful":   len(imported),
		"skipped":      len(skipped),
		"failed":       0,
		"imported":     imported,
		"skipped_rows": skipped,
	}, false, "")
}

func (c *LocationsController) ValidateImportRows(ctx *gin.Context) {
	var rows []requests.LocationImportRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		tools.ResponseBadRequest(ctx, "ValidateImportRows", "JSON inválido", "validate_location_import_rows")
		return
	}
	if len(rows) == 0 {
		tools.ResponseBadRequest(ctx, "ValidateImportRows", "No se proporcionaron filas", "validate_location_import_rows")
		return
	}
	results, resp := c.Service.ValidateImportRows(c.resolveTenantID(ctx), rows)
	if resp != nil {
		writeErrorResponse(ctx, "ValidateImportRows", "validate_location_import_rows", resp)
		return
	}
	tools.ResponseOK(ctx, "ValidateImportRows", "Validación completada", "validate_location_import_rows", gin.H{
		"results": results,
	}, false, "")
}

func (c *LocationsController) ImportLocationsFromJSON(ctx *gin.Context) {
	var rows []requests.LocationImportRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		tools.ResponseBadRequest(ctx, "ImportLocationsFromJSON", "JSON inválido", "import_locations_from_json")
		return
	}
	if len(rows) == 0 {
		tools.ResponseBadRequest(ctx, "ImportLocationsFromJSON", "No se proporcionaron filas", "import_locations_from_json")
		return
	}
	imported, skipped, errResp := c.Service.ImportLocationsFromJSON(c.resolveTenantID(ctx), rows)
	if errResp != nil {
		writeErrorResponse(ctx, "ImportLocationsFromJSON", "import_locations_from_json", errResp)
		return
	}
	tools.ResponseOK(ctx, "ImportLocationsFromJSON", "Importación completada", "import_locations_from_json", gin.H{
		"successful":   len(imported),
		"skipped":      len(skipped),
		"failed":       0,
		"imported":     imported,
		"skipped_rows": skipped,
	}, false, "")
}

func (c *LocationsController) DownloadImportTemplate(ctx *gin.Context) {
	lang := ctx.DefaultQuery("lang", "es")
	data, err := c.Service.GenerateImportTemplate(lang)
	if err != nil {
		tools.ResponseBadRequest(ctx, "DownloadImportTemplate", "Error al generar la plantilla", "download_import_template")
		return
	}
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="ImportLocations.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (c *LocationsController) ExportLocationsToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportLocationsToExcel(c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "ExportLocationsToExcel", "export_locations_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="locations.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *LocationsController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
