package repositories

import (
	"context"
	"errors"
	"strconv"

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
type LocationsRepositorySQLC struct {
	queries *sqlc.Queries
	gorm    *LocationsRepository
}

// NewLocationsRepositorySQLC returns a locations repository backed by sqlc; Excel uses gorm.
func NewLocationsRepositorySQLC(queries *sqlc.Queries, gorm *LocationsRepository) *LocationsRepositorySQLC {
	return &LocationsRepositorySQLC{queries: queries, gorm: gorm}
}

var _ ports.LocationsRepository = (*LocationsRepositorySQLC)(nil)

func (r *LocationsRepositorySQLC) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListLocations(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las ubicaciones", Handled: false}
	}
	out := make([]database.Location, len(list))
	for i, loc := range list {
		out[i] = sqlcLocationToDatabase(loc)
	}
	return out, nil
}

func (r *LocationsRepositorySQLC) GetLocationByID(id string) (*database.Location, *responses.InternalResponse) {
	ctx := context.Background()
	idInt, err := strconv.Atoi(id)
	if err != nil {
		// Try as location_code
		loc, err := r.queries.GetLocationByLocationCode(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, &responses.InternalResponse{
					Message:    "Ubicación no encontrada",
					Handled:    true,
					StatusCode: responses.StatusNotFound,
				}
			}
			return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la ubicación", Handled: false}
		}
		l := sqlcLocationToDatabase(loc)
		return &l, nil
	}
	loc, err := r.queries.GetLocationByID(ctx, int32(idInt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Ubicación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la ubicación", Handled: false}
	}
	l := sqlcLocationToDatabase(loc)
	return &l, nil
}

func (r *LocationsRepositorySQLC) CreateLocation(input *requests.Location) *responses.InternalResponse {
	ctx := context.Background()
	exists, err := r.queries.LocationExistsByLocationCode(ctx, input.LocationCode)
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
	}
	_, err = r.queries.CreateLocation(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) UpdateLocation(id int, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	loc, err := r.queries.GetLocationByID(ctx, int32(id))
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
	arg := sqlc.UpdateLocationParams{
		ID:           loc.ID,
		LocationCode: loc.LocationCode,
		Description:  loc.Description,
		Zone:         loc.Zone,
		Type:         loc.Type,
		IsActive:     loc.IsActive,
	}
	_, err = r.queries.UpdateLocation(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) DeleteLocation(id int) *responses.InternalResponse {
	ctx := context.Background()
	_, err := r.queries.GetLocationByID(ctx, int32(id))
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
	if err := r.queries.DeleteLocation(ctx, int32(id)); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la ubicación", Handled: false}
	}
	return nil
}

func (r *LocationsRepositorySQLC) ImportLocationsFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return r.gorm.ImportLocationsFromExcel(fileBytes)
}

func (r *LocationsRepositorySQLC) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	return r.gorm.ExportLocationsToExcel()
}

func sqlcLocationToDatabase(l sqlc.Location) database.Location {
	return database.Location{
		ID:           int(l.ID),
		LocationCode: l.LocationCode,
		Description:  pgTextToPtrString(l.Description),
		Zone:         pgTextToPtrString(l.Zone),
		Type:         l.Type,
		IsActive:     l.IsActive,
		CreatedAt:    pgTimestampToTime(l.CreatedAt),
		UpdatedAt:    pgTimestampToTime(l.UpdatedAt),
	}
}
