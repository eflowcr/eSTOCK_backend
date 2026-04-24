package repositories

import (
	"context"
	"errors"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LocationsRepositorySQLC implements ports.LocationsRepository using sqlc for CRUD.
// Excel import/export are delegated to the GORM fallback repo.
//
// S3.5 W2-A: every method is tenant-scoped — the SQL queries enforce
// WHERE tenant_id = $N and INSERT INTO ... (..., tenant_id) VALUES (..., $N).
type LocationsRepositorySQLC struct {
	queries *sqlc.Queries
	gorm    *LocationsRepository
}

// NewLocationsRepositorySQLC returns a locations repository backed by sqlc; Excel uses gorm.
func NewLocationsRepositorySQLC(queries *sqlc.Queries, gorm *LocationsRepository) *LocationsRepositorySQLC {
	return &LocationsRepositorySQLC{queries: queries, gorm: gorm}
}

var _ ports.LocationsRepository = (*LocationsRepositorySQLC)(nil)

func (r *LocationsRepositorySQLC) GetAllLocations(tenantID string) ([]database.Location, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	list, err := r.queries.ListLocationsByTenant(ctx, tid)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las ubicaciones", Handled: false}
	}
	out := make([]database.Location, len(list))
	for i, loc := range list {
		out[i] = locationRowToDatabase(loc.ID, loc.LocationCode, loc.Description, loc.Zone, loc.Type, loc.IsActive, loc.IsWayOut, loc.CreatedAt, loc.UpdatedAt, loc.TenantID)
	}
	return out, nil
}

func (r *LocationsRepositorySQLC) GetLocationByID(tenantID, id string) (*database.Location, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	loc, err := r.queries.GetLocationByIDForTenant(ctx, sqlc.GetLocationByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Backward-compat fallback: caller may have passed a location_code.
			loc2, err2 := r.queries.GetLocationByLocationCodeForTenant(ctx, sqlc.GetLocationByLocationCodeForTenantParams{LocationCode: id, TenantID: tid})
			if err2 == nil {
				l := locationRowToDatabase(loc2.ID, loc2.LocationCode, loc2.Description, loc2.Zone, loc2.Type, loc2.IsActive, loc2.IsWayOut, loc2.CreatedAt, loc2.UpdatedAt, loc2.TenantID)
				return &l, nil
			}
			if errors.Is(err2, pgx.ErrNoRows) {
				return nil, &responses.InternalResponse{
					Message:    "Ubicación no encontrada",
					Handled:    true,
					StatusCode: responses.StatusNotFound,
				}
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la ubicación", Handled: false}
	}
	l := locationRowToDatabase(loc.ID, loc.LocationCode, loc.Description, loc.Zone, loc.Type, loc.IsActive, loc.IsWayOut, loc.CreatedAt, loc.UpdatedAt, loc.TenantID)
	return &l, nil
}

func (r *LocationsRepositorySQLC) CreateLocation(tenantID string, input *requests.Location) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	exists, err := r.queries.LocationExistsByLocationCodeForTenant(ctx, sqlc.LocationExistsByLocationCodeForTenantParams{LocationCode: input.LocationCode, TenantID: tid})
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al verificar la existencia de la ubicación", Handled: false}
	}
	if exists {
		return &responses.InternalResponse{
			Message: "El código de ubicación ya existe",
			Handled: true,
		}
	}
	arg := sqlc.CreateLocationParams{
		LocationCode: input.LocationCode,
		Description:  ptrStringToPgText(input.Description),
		Zone:         ptrStringToPgText(input.Zone),
		Type:         input.Type,
		IsActive:     true,
		IsWayOut:     input.IsWayOut,
		TenantID:     tid,
	}
	_, err = r.queries.CreateLocation(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) UpdateLocation(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	loc, err := r.queries.GetLocationByIDForTenant(ctx, sqlc.GetLocationByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Ubicación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al obtener la ubicación", Handled: false}
	}
	// Merge allowed fields from data
	if v, ok := data["location_code"].(string); ok {
		loc.LocationCode = v
	}
	if v, ok := data["description"].(*string); ok && v != nil {
		loc.Description = pgtype.Text{String: *v, Valid: true}
	}
	if v, ok := data["zone"].(*string); ok && v != nil {
		loc.Zone = pgtype.Text{String: *v, Valid: true}
	}
	if v, ok := data["type"].(string); ok {
		loc.Type = v
	}
	if v, ok := data["is_active"].(bool); ok {
		loc.IsActive = v
	}
	if v, ok := data["is_way_out"].(bool); ok {
		loc.IsWayOut = v
	}
	arg := sqlc.UpdateLocationForTenantParams{
		ID:           loc.ID,
		LocationCode: loc.LocationCode,
		Description:  loc.Description,
		Zone:         loc.Zone,
		Type:         loc.Type,
		IsActive:     loc.IsActive,
		IsWayOut:     loc.IsWayOut,
		TenantID:     tid,
	}
	_, err = r.queries.UpdateLocationForTenant(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) DeleteLocation(tenantID, id string) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	_, err = r.queries.GetLocationByIDForTenant(ctx, sqlc.GetLocationByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Ubicación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al obtener la ubicación", Handled: false}
	}
	if err := r.queries.DeleteLocationForTenant(ctx, sqlc.DeleteLocationForTenantParams{ID: id, TenantID: tid}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) ImportLocationsFromExcel(tenantID string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return r.gorm.ImportLocationsFromExcel(tenantID, fileBytes)
}

func (r *LocationsRepositorySQLC) ImportLocationsFromJSON(tenantID string, rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	return r.gorm.ImportLocationsFromJSON(tenantID, rows)
}

func (r *LocationsRepositorySQLC) ValidateImportRows(tenantID string, rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	return r.gorm.ValidateImportRows(tenantID, rows)
}

func (r *LocationsRepositorySQLC) ExportLocationsToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	return r.gorm.ExportLocationsToExcel(tenantID)
}

func locationRowToDatabase(id, locationCode string, description, zone pgtype.Text, locType string, isActive, isWayOut bool, createdAt, updatedAt pgtype.Timestamp, tenantID pgtype.UUID) database.Location {
	return database.Location{
		ID:           id,
		TenantID:     pgUUIDToString(tenantID),
		LocationCode: locationCode,
		Description:  pgTextToPtrString(description),
		Zone:         pgTextToPtrString(zone),
		Type:         locType,
		IsActive:     isActive,
		IsWayOut:     isWayOut,
		CreatedAt:    pgTimestampToTime(createdAt),
		UpdatedAt:    pgTimestampToTime(updatedAt),
	}
}

func (r *LocationsRepositorySQLC) GenerateImportTemplate(language string) ([]byte, error) {
	return r.gorm.GenerateImportTemplate(language)
}
