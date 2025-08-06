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
			Message: "Failed to fetch locations",
			Handled: false,
		}
	}

	if len(locations) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No locations found",
			Handled: true,
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
				Error:   nil,
				Message: "Location not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch location",
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
			Message: "Location code already exists",
			Handled: true,
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check existing location",
			Handled: false,
		}
	}

	location := &database.Location{
		LocationCode: input.LocationCode,
		Description:  input.Description,
		Zone:         input.Zone,
		Type:         input.Type,
		IsActive:     true,
		CreatedAt:    tools.GetCurrentTime(),
		UpdatedAt:    tools.GetCurrentTime(),
	}

	err = r.DB.Create(location).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create location",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) UpdateLocation(id int, data map[string]interface{}) *responses.InternalResponse {
	var location database.Location
	err := r.DB.First(&location, "id = ?", id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Location not found",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to retrieve location",
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
			Message: "Failed to update location",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) DeleteLocation(id int) *responses.InternalResponse {
	var location database.Location
	err := r.DB.First(&location, "id = ?", id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Location not found",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to retrieve location",
			Handled: false,
		}
	}

	err = r.DB.Delete(&location).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to delete location",
			Handled: false,
		}
	}

	return nil
}

func (r *LocationsRepository) ImportLocationsFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	imported := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to open Excel file",
			Handled: false,
		})
		return imported, errorsList
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to read rows",
			Handled: false,
		})
		return imported, errorsList
	}

	for i, row := range rows {
		if i < 6 { // Saltar encabezado
			continue
		}

		if len(row) < 4 {
			continue
		}

		locationCode := strings.TrimSpace(row[0])
		description := strings.TrimSpace(row[1])
		zone := strings.TrimSpace(row[2])
		locType := strings.TrimSpace(row[3])

		if locationCode == "" || locType == "" {
			continue
		}

		descPtr := &description
		if description == "" {
			descPtr = nil
		}

		zonePtr := &zone
		if zone == "" {
			zonePtr = nil
		}

		loc := &requests.Location{
			LocationCode: locationCode,
			Description:  descPtr,
			Zone:         zonePtr,
			Type:         locType,
		}

		resp := r.CreateLocation(loc)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Row %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}

		imported = append(imported, locationCode)
	}

	return imported, errorsList
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
			Message: "Failed to generate Excel file",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func getOrEmpty(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
