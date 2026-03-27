package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type LocationsRepository struct {
	DB *gorm.DB
}

func (r *LocationsRepository) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	var locations []database.Location

	err := r.DB.
		Table(database.Location{}.TableName()).
		Order("created_at ASC").
		Find(&locations).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener las ubicaciones",
			Handled: false,
		}
	}

	return locations, nil
}

func (r *LocationsRepository) GetLocationByID(id string) (*database.Location, *responses.InternalResponse) {
	var location database.Location

	err := r.DB.
		Table(database.Location{}.TableName()).
		Where("id = ?", id).
		First(&location).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Ubicación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la ubicación",
			Handled: false,
		}
	}

	return &location, nil
}

func (r *LocationsRepository) CreateLocation(input *requests.Location) *responses.InternalResponse {
	// Verificar si el código ya existe
	var existing database.Location
	err := r.DB.First(&existing, "location_code = ?", input.LocationCode).Error

	if err == nil {
		return &responses.InternalResponse{
			Message: "El código de ubicación ya existe",
			Handled: true,
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al verificar la existencia de la ubicación",
			Handled: false,
		}
	}

	location := &database.Location{
		LocationCode: input.LocationCode,
		Description:  input.Description,
		Zone:         input.Zone,
		Type:         input.Type,
		IsActive:     true,
		IsWayOut:     input.IsWayOut,
		CreatedAt:    tools.GetCurrentTime(),
		UpdatedAt:    tools.GetCurrentTime(),
	}

	err = r.DB.Create(location).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear la ubicación",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) UpdateLocation(id string, data map[string]interface{}) *responses.InternalResponse {
	var location database.Location
	err := r.DB.First(&location, "id = ?", id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Message:    "Ubicación no encontrada",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la ubicación",
			Handled: false,
		}
	}

	protectedFields := map[string]bool{
		"id":         true,
		"created_at": true,
	}

	for k := range protectedFields {
		delete(data, k)
	}

	data["updated_at"] = tools.GetCurrentTime()

	if err := r.DB.Model(&location).Updates(data).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar la ubicación",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) DeleteLocation(id string) *responses.InternalResponse {
	var location database.Location
	err := r.DB.First(&location, "id = ?", id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Message:    "Ubicación no encontrada",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la ubicación",
			Handled: false,
		}
	}

	err = r.DB.Delete(&location).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar la ubicación",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) ImportLocationsFromExcel(fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return imported, skipped, &responses.InternalResponse{Error: err, Message: "Error al abrir el archivo", Handled: false}
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return imported, skipped, &responses.InternalResponse{Message: "Sin hojas de datos", Handled: true}
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return imported, skipped, &responses.InternalResponse{Error: err, Message: "Error al leer filas", Handled: false}
	}

	for i, row := range rows {
		if i < 8 { // Skip header, instructions, col headers, example (rows 1-8)
			continue
		}
		if len(row) < 4 {
			continue
		}

		locationCode := strings.TrimSpace(row[0])
		locType := strings.TrimSpace(row[3])

		if locationCode == "" || locType == "" {
			continue
		}

		// Skip example row
		if strings.EqualFold(locationCode, "LOC-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
			continue
		}

		rowReq := requests.LocationImportRow{
			LocationCode: locationCode,
			Description:  strings.TrimSpace(row[1]),
			Zone:         strings.TrimSpace(row[2]),
			Type:         locType,
		}

		imp, sk, errResp := r.ImportLocationsFromJSON([]requests.LocationImportRow{rowReq})
		imported = append(imported, imp...)
		skipped = append(skipped, sk...)
		if errResp != nil {
			return imported, skipped, errResp
		}
	}

	return imported, skipped, nil
}

func (r *LocationsRepository) ImportLocationsFromJSON(rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}

	for i, row := range rows {
		code := strings.TrimSpace(row.LocationCode)
		locType := strings.TrimSpace(row.Type)

		if code == "" || locType == "" {
			skipped = append(skipped, fmt.Sprintf("Fila %d: código y tipo son requeridos", i+1))
			continue
		}
		if strings.EqualFold(code, "LOC-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
			continue
		}

		desc := strings.TrimSpace(row.Description)
		zone := strings.TrimSpace(row.Zone)
		var descPtr, zonePtr *string
		if desc != "" {
			descPtr = &desc
		}
		if zone != "" {
			zonePtr = &zone
		}

		loc := &requests.Location{LocationCode: code, Description: descPtr, Zone: zonePtr, Type: locType}
		resp := r.CreateLocation(loc)
		if resp != nil {
			return imported, skipped, &responses.InternalResponse{
				Error: resp.Error, Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message), Handled: resp.Handled,
			}
		}
		imported = append(imported, code)
	}

	return imported, skipped, nil
}

func (r *LocationsRepository) ValidateImportRows(rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	results := make([]responses.LocationValidationResult, 0, len(rows))
	seenCodes := make(map[string]bool)

	for i, row := range rows {
		code := strings.TrimSpace(row.LocationCode)
		locType := strings.TrimSpace(row.Type)
		result := responses.LocationValidationResult{RowIndex: i, Row: row}

		// Field validation
		if code == "" || locType == "" {
			result.Status = responses.LocationStatusError
			result.FieldErrors = map[string]string{}
			if code == "" {
				result.FieldErrors["location_code"] = "Código requerido"
			}
			if locType == "" {
				result.FieldErrors["type"] = "Tipo requerido"
			}
			results = append(results, result)
			continue
		}

		// Duplicate within batch
		if seenCodes[strings.ToLower(code)] {
			result.Status = responses.LocationStatusDuplicate
			results = append(results, result)
			continue
		}
		seenCodes[strings.ToLower(code)] = true

		// Exact code match in DB
		var existing database.Location
		if err := r.DB.Where("location_code = ?", code).First(&existing).Error; err == nil {
			result.Status = responses.LocationStatusExists
			result.ExistingLocation = &responses.LocationValidationMatch{
				ID:           existing.ID,
				LocationCode: existing.LocationCode,
				Type:         existing.Type,
				IsActive:     existing.IsActive,
			}
			if existing.Description != nil {
				result.ExistingLocation.Description = *existing.Description
			}
			if existing.Zone != nil {
				result.ExistingLocation.Zone = *existing.Zone
			}
			results = append(results, result)
			continue
		}

		// Similar description check
		desc := strings.TrimSpace(row.Description)
		if desc != "" {
			keyword := desc
			if len(keyword) > 20 {
				keyword = keyword[:20]
			}
			var similar []database.Location
			r.DB.Where("LOWER(description) LIKE LOWER(?) AND location_code != ?", "%"+keyword+"%", code).Limit(3).Find(&similar)
			if len(similar) > 0 {
				result.Status = responses.LocationStatusSimilar
				for _, s := range similar {
					m := responses.LocationValidationMatch{ID: s.ID, LocationCode: s.LocationCode, Type: s.Type, IsActive: s.IsActive}
					if s.Description != nil {
						m.Description = *s.Description
					}
					result.SimilarLocations = append(result.SimilarLocations, m)
				}
				results = append(results, result)
				continue
			}
		}

		result.Status = responses.LocationStatusNew
		results = append(results, result)
	}

	return results, nil
}

func (l *LocationsRepository) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	locations, errResp := l.GetAllLocations()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Descripción", "Zona", "Tipo"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, loc := range locations {
		row := idx + 7
		values := []interface{}{
			loc.ID,
			getOrEmpty(loc.Description),
			getOrEmpty(loc.Zone),
			loc.Type,
		}
		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar el archivo de Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func getOrEmpty[T any](ptr *T) interface{} {
	if ptr == nil {
		return ""
	}
	return fmt.Sprintf("%v", *ptr)
}

func (l *LocationsRepository) GenerateImportTemplate(language string) ([]byte, error) {
	isEs := language != "en"
	typeOpts := []string{"PALLET", "SHELF", "BIN", "FLOOR", "BLOCK"}

	cfg := ModuleTemplateConfig{
		DataSheetName: ifStr(isEs, "Ubicaciones", "Locations"),
		OptSheetName:  ifStr(isEs, "Opciones", "Options"),
		Title:         ifStr(isEs, "Importar Ubicaciones", "Import Locations"),
		Subtitle:      ifStr(isEs, "Plantilla de importación — eSTOCK", "Location import template — eSTOCK"),
		InstrTitle:    ifStr(isEs, "📋 Instrucciones", "📋 Instructions"),
		InstrContent: ifStr(isEs,
			"1. Complete desde la fila 9  •  2. El campo Tipo acepta solo valores de la lista desplegable  •  3. ID y Descripción son obligatorios (*)",
			"1. Fill in data from row 9 onwards  •  2. Type field accepts only values from the dropdown  •  3. ID and Description are required (*)"),
		LogoOffsetX: 55,
		LogoOffsetY: 8,
		Columns: []ColumnDef{
			{Header: "ID *", Required: true, Width: 16},
			{Header: ifStr(isEs, "Descripción *", "Description *"), Required: true, Width: 32},
			{Header: ifStr(isEs, "Zona", "Zone"), Required: false, Width: 20},
			{Header: ifStr(isEs, "Tipo", "Type"), Required: false, Width: 16},
		},
		ExampleRow: []string{
			"LOC-001",
			ifStr(isEs, "Rack Principal Zona A", "Main Rack Zone A"),
			"Zone A",
			"SHELF",
		},
		ApplyValidations: func(f *excelize.File, dataSheet, optSheet string, start, end int) error {
			// ── Logo area: re-merge A1:D2 as one region + light blue background ──
			f.UnmergeCell(dataSheet, "A1", "D1")
			f.UnmergeCell(dataSheet, "A2", "D2")
			f.MergeCell(dataSheet, "A1", "D2")
			logoAreaStyle, _ := f.NewStyle(&excelize.Style{
				Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EEF2FF"}},
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			})
			f.SetCellStyle(dataSheet, "A1", "D2", logoAreaStyle)

			// ── Dropdown options sheet ──────────────────────────────────────────
			f.NewSheet(optSheet)
			for i, v := range typeOpts {
				cell, _ := excelize.CoordinatesToCellName(1, i+1)
				f.SetCellValue(optSheet, cell, v)
			}
			f.SetSheetVisible(optSheet, false)
			return SharedDropListValidation(f, dataSheet, optSheet,
				"D9:D2000", "$A$1:$A$5",
				ifStr(isEs, "Tipo inválido", "Invalid type"),
				ifStr(isEs, "Seleccione un tipo de la lista", "Select a type from the list"),
			)
		},
	}
	return BuildModuleImportTemplate(cfg)
}
