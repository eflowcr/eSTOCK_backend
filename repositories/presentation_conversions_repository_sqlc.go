package repositories

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PresentationConversionsRepositorySQLC struct {
	queries *sqlc.Queries
}

func NewPresentationConversionsRepositorySQLC(queries *sqlc.Queries) *PresentationConversionsRepositorySQLC {
	return &PresentationConversionsRepositorySQLC{queries: queries}
}

var _ ports.PresentationConversionsRepository = (*PresentationConversionsRepositorySQLC)(nil)

func (r *PresentationConversionsRepositorySQLC) ListPresentationConversions() ([]database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListPresentationConversions(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing presentation conversions", Handled: false}
	}
	out := make([]database.PresentationConversion, len(list))
	for i, row := range list {
		out[i] = sqlcConversionToDatabase(row)
	}
	return out, nil
}

func (r *PresentationConversionsRepositorySQLC) ListPresentationConversionsAdmin() ([]database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListPresentationConversionsAdmin(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing presentation conversions (admin)", Handled: false}
	}
	out := make([]database.PresentationConversion, len(list))
	for i, row := range list {
		out[i] = sqlcConversionToDatabase(row)
	}
	return out, nil
}

func (r *PresentationConversionsRepositorySQLC) GetPresentationConversionByID(id string) (*database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetPresentationConversionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Presentation conversion not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation conversion", Handled: false}
	}
	out := sqlcConversionToDatabase(row)
	return &out, nil
}

func (r *PresentationConversionsRepositorySQLC) GetPresentationConversionByFromAndTo(fromID, toID string) (*database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetPresentationConversionByFromAndTo(ctx, sqlc.GetPresentationConversionByFromAndToParams{
		FromPresentationTypeID: fromID,
		ToPresentationTypeID:  toID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation conversion", Handled: false}
	}
	out := sqlcConversionToDatabase(row)
	return &out, nil
}

func (r *PresentationConversionsRepositorySQLC) CreatePresentationConversion(req *requests.PresentationConversionCreate) (*database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	existing, resp := r.GetPresentationConversionByFromAndTo(req.FromPresentationTypeID, req.ToPresentationTypeID)
	if resp != nil {
		return nil, resp
	}
	if existing != nil {
		return nil, &responses.InternalResponse{Message: "Conversion rule for this from/to pair already exists", Handled: true, StatusCode: responses.StatusConflict}
	}
	if req.FromPresentationTypeID == req.ToPresentationTypeID {
		return nil, &responses.InternalResponse{Message: "From and to presentation type must be different", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.CreatePresentationConversionParams{
		FromPresentationTypeID: req.FromPresentationTypeID,
		ToPresentationTypeID:   req.ToPresentationTypeID,
		ConversionFactor:       floatToPgNumeric(req.ConversionFactor),
		IsActive:               isActive,
	}
	row, err := r.queries.CreatePresentationConversion(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating presentation conversion", Handled: false}
	}
	out := sqlcConversionToDatabase(row)
	return &out, nil
}

func (r *PresentationConversionsRepositorySQLC) UpdatePresentationConversion(id string, req *requests.PresentationConversionUpdate) (*database.PresentationConversion, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetPresentationConversionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Presentation conversion not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting presentation conversion", Handled: false}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.UpdatePresentationConversionParams{
		ID:               id,
		ConversionFactor: floatToPgNumeric(req.ConversionFactor),
		IsActive:         isActive,
	}
	row, err := r.queries.UpdatePresentationConversion(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating presentation conversion", Handled: false}
	}
	out := sqlcConversionToDatabase(row)
	return &out, nil
}

func (r *PresentationConversionsRepositorySQLC) DeletePresentationConversion(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeletePresentationConversion(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting presentation conversion", Handled: false}
	}
	return nil
}

func floatToPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

func sqlcConversionToDatabase(row sqlc.PresentationConversion) database.PresentationConversion {
	cAt := time.Time{}
	uAt := time.Time{}
	if row.CreatedAt.Valid {
		cAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		uAt = row.UpdatedAt.Time
	}
	return database.PresentationConversion{
		ID:                     row.ID,
		FromPresentationTypeID: row.FromPresentationTypeID,
		ToPresentationTypeID:   row.ToPresentationTypeID,
		ConversionFactor:         pgNumericToFloat(row.ConversionFactor),
		IsActive:                row.IsActive,
		CreatedAt:               cAt,
		UpdatedAt:               uAt,
	}
}
