package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
)

type LocationTypesRepositorySQLC struct {
	queries *sqlc.Queries
}

func NewLocationTypesRepositorySQLC(queries *sqlc.Queries) *LocationTypesRepositorySQLC {
	return &LocationTypesRepositorySQLC{queries: queries}
}

var _ ports.LocationTypesRepository = (*LocationTypesRepositorySQLC)(nil)

func (r *LocationTypesRepositorySQLC) ListLocationTypes() ([]database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListLocationTypes(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing location types", Handled: false}
	}
	out := make([]database.LocationType, len(list))
	for i, row := range list {
		out[i] = sqlcLocationTypeToDatabase(row)
	}
	return out, nil
}

func (r *LocationTypesRepositorySQLC) ListLocationTypesAdmin() ([]database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListLocationTypesAdmin(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing location types (admin)", Handled: false}
	}
	out := make([]database.LocationType, len(list))
	for i, row := range list {
		out[i] = sqlcLocationTypeToDatabase(row)
	}
	return out, nil
}

func (r *LocationTypesRepositorySQLC) GetLocationTypeByID(id string) (*database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetLocationTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Location type not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting location type", Handled: false}
	}
	out := sqlcLocationTypeToDatabase(row)
	return &out, nil
}

func (r *LocationTypesRepositorySQLC) GetLocationTypeByCode(code string) (*database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetLocationTypeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting location type by code", Handled: false}
	}
	out := sqlcLocationTypeToDatabase(row)
	return &out, nil
}

func (r *LocationTypesRepositorySQLC) CreateLocationType(req *requests.LocationTypeCreate) (*database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	exists, err := r.queries.LocationTypeExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error checking location type code", Handled: false}
	}
	if exists {
		return nil, &responses.InternalResponse{Message: "Location type with this code already exists", Handled: true, StatusCode: responses.StatusConflict}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.CreateLocationTypeParams{
		Code:      req.Code,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  isActive,
	}
	row, err := r.queries.CreateLocationType(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating location type", Handled: false}
	}
	out := sqlcLocationTypeToDatabase(row)
	return &out, nil
}

func (r *LocationTypesRepositorySQLC) UpdateLocationType(id string, req *requests.LocationTypeUpdate) (*database.LocationType, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetLocationTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Location type not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting location type", Handled: false}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.UpdateLocationTypeParams{
		ID:        id,
		Code:      req.Code,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  isActive,
	}
	row, err := r.queries.UpdateLocationType(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating location type", Handled: false}
	}
	out := sqlcLocationTypeToDatabase(row)
	return &out, nil
}

func (r *LocationTypesRepositorySQLC) DeleteLocationType(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeleteLocationType(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting location type", Handled: false}
	}
	return nil
}

func sqlcLocationTypeToDatabase(row sqlc.LocationType) database.LocationType {
	cAt := time.Time{}
	uAt := time.Time{}
	if row.CreatedAt.Valid {
		cAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		uAt = row.UpdatedAt.Time
	}
	return database.LocationType{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		SortOrder: row.SortOrder,
		IsActive:  row.IsActive,
		CreatedAt: cAt,
		UpdatedAt: uAt,
	}
}
