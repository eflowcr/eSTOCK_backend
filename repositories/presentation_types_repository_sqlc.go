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

type PresentationTypesRepositorySQLC struct {
	queries *sqlc.Queries
}

func NewPresentationTypesRepositorySQLC(queries *sqlc.Queries) *PresentationTypesRepositorySQLC {
	return &PresentationTypesRepositorySQLC{queries: queries}
}

var _ ports.PresentationTypesRepository = (*PresentationTypesRepositorySQLC)(nil)

func (r *PresentationTypesRepositorySQLC) ListPresentationTypes() ([]database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListPresentationTypes(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing presentation types", Handled: false}
	}
	out := make([]database.PresentationType, len(list))
	for i, row := range list {
		out[i] = sqlcPresentationTypeToDatabase(row)
	}
	return out, nil
}

func (r *PresentationTypesRepositorySQLC) ListPresentationTypesAdmin() ([]database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListPresentationTypesAdmin(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing presentation types (admin)", Handled: false}
	}
	out := make([]database.PresentationType, len(list))
	for i, row := range list {
		out[i] = sqlcPresentationTypeToDatabase(row)
	}
	return out, nil
}

func (r *PresentationTypesRepositorySQLC) GetPresentationTypeByID(id string) (*database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetPresentationTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Presentation type not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation type", Handled: false}
	}
	out := sqlcPresentationTypeToDatabase(row)
	return &out, nil
}

func (r *PresentationTypesRepositorySQLC) GetPresentationTypeByCode(code string) (*database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetPresentationTypeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation type by code", Handled: false}
	}
	out := sqlcPresentationTypeToDatabase(row)
	return &out, nil
}

func (r *PresentationTypesRepositorySQLC) CreatePresentationType(req *requests.PresentationTypeCreate) (*database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	exists, err := r.queries.PresentationTypeExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error checking presentation type code", Handled: false}
	}
	if exists {
		return nil, &responses.InternalResponse{Message: "Presentation type with this code already exists", Handled: true, StatusCode: responses.StatusConflict}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.CreatePresentationTypeParams{
		Code:      req.Code,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  isActive,
	}
	row, err := r.queries.CreatePresentationType(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating presentation type", Handled: false}
	}
	out := sqlcPresentationTypeToDatabase(row)
	return &out, nil
}

func (r *PresentationTypesRepositorySQLC) UpdatePresentationType(id string, req *requests.PresentationTypeUpdate) (*database.PresentationType, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetPresentationTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Presentation type not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation type", Handled: false}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.UpdatePresentationTypeParams{
		ID:        id,
		Code:      req.Code,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  isActive,
	}
	row, err := r.queries.UpdatePresentationType(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating presentation type", Handled: false}
	}
	out := sqlcPresentationTypeToDatabase(row)
	return &out, nil
}

func (r *PresentationTypesRepositorySQLC) DeletePresentationType(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeletePresentationType(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting presentation type", Handled: false}
	}
	return nil
}

func sqlcPresentationTypeToDatabase(row sqlc.PresentationType) database.PresentationType {
	cAt := time.Time{}
	uAt := time.Time{}
	if row.CreatedAt.Valid {
		cAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		uAt = row.UpdatedAt.Time
	}
	return database.PresentationType{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		SortOrder: row.SortOrder,
		IsActive:  row.IsActive,
		CreatedAt: cAt,
		UpdatedAt: uAt,
	}
}
