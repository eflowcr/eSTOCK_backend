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

type AdjustmentReasonCodesRepositorySQLC struct {
	queries *sqlc.Queries
}

func NewAdjustmentReasonCodesRepositorySQLC(queries *sqlc.Queries) *AdjustmentReasonCodesRepositorySQLC {
	return &AdjustmentReasonCodesRepositorySQLC{queries: queries}
}

var _ ports.AdjustmentReasonCodesRepository = (*AdjustmentReasonCodesRepositorySQLC)(nil)

func (r *AdjustmentReasonCodesRepositorySQLC) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListAdjustmentReasonCodes(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing adjustment reason codes", Handled: false}
	}
	out := make([]database.AdjustmentReasonCode, len(list))
	for i, row := range list {
		out[i] = sqlcAdjustmentReasonCodeToDatabase(row)
	}
	return out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListAdjustmentReasonCodesAdmin(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing adjustment reason codes (admin)", Handled: false}
	}
	out := make([]database.AdjustmentReasonCode, len(list))
	for i, row := range list {
		out[i] = sqlcAdjustmentReasonCodeToDatabase(row)
	}
	return out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetAdjustmentReasonCodeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Adjustment reason code not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting adjustment reason code", Handled: false}
	}
	out := sqlcAdjustmentReasonCodeToDatabase(row)
	return &out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetAdjustmentReasonCodeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting adjustment reason code by code", Handled: false}
	}
	out := sqlcAdjustmentReasonCodeToDatabase(row)
	return &out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	existing, _ := r.GetAdjustmentReasonCodeByCode(req.Code)
	if existing != nil {
		return nil, &responses.InternalResponse{Message: "Adjustment reason code with this code already exists", Handled: true, StatusCode: responses.StatusConflict}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.CreateAdjustmentReasonCodeParams{
		Code:         req.Code,
		Name:         req.Name,
		Direction:    req.Direction,
		IsSystem:     false,
		DisplayOrder: req.DisplayOrder,
		IsActive:     isActive,
	}
	row, err := r.queries.CreateAdjustmentReasonCode(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating adjustment reason code", Handled: false}
	}
	out := sqlcAdjustmentReasonCodeToDatabase(row)
	return &out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	ctx := context.Background()
	existing, resp := r.GetAdjustmentReasonCodeByID(id)
	if resp != nil {
		return nil, resp
	}
	if existing == nil {
		return nil, &responses.InternalResponse{Message: "Adjustment reason code not found", Handled: true, StatusCode: responses.StatusNotFound}
	}
	if existing.Code != req.Code {
		other, _ := r.GetAdjustmentReasonCodeByCode(req.Code)
		if other != nil {
			return nil, &responses.InternalResponse{Message: "Another reason code with this code already exists", Handled: true, StatusCode: responses.StatusConflict}
		}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	arg := sqlc.UpdateAdjustmentReasonCodeParams{
		ID:           id,
		Code:         req.Code,
		Name:         req.Name,
		Direction:    req.Direction,
		DisplayOrder: req.DisplayOrder,
		IsActive:     isActive,
	}
	row, err := r.queries.UpdateAdjustmentReasonCode(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating adjustment reason code", Handled: false}
	}
	out := sqlcAdjustmentReasonCodeToDatabase(row)
	return &out, nil
}

func (r *AdjustmentReasonCodesRepositorySQLC) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	ctx := context.Background()
	existing, resp := r.GetAdjustmentReasonCodeByID(id)
	if resp != nil {
		return resp
	}
	if existing == nil {
		return &responses.InternalResponse{Message: "Adjustment reason code not found", Handled: true, StatusCode: responses.StatusNotFound}
	}
	if existing.IsSystem {
		return &responses.InternalResponse{Message: "System reason codes cannot be deleted", Handled: true, StatusCode: responses.StatusForbidden}
	}
	err := r.queries.DeleteAdjustmentReasonCode(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting adjustment reason code", Handled: false}
	}
	return nil
}

func sqlcAdjustmentReasonCodeToDatabase(row sqlc.AdjustmentReasonCode) database.AdjustmentReasonCode {
	cAt := time.Time{}
	uAt := time.Time{}
	if row.CreatedAt.Valid {
		cAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		uAt = row.UpdatedAt.Time
	}
	return database.AdjustmentReasonCode{
		ID:           row.ID,
		Code:         row.Code,
		Name:         row.Name,
		Direction:    row.Direction,
		IsSystem:     row.IsSystem,
		DisplayOrder: row.DisplayOrder,
		IsActive:     row.IsActive,
		CreatedAt:    cAt,
		UpdatedAt:    uAt,
	}
}
