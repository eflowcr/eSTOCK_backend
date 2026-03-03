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
	"github.com/jackc/pgx/v5/pgtype"
)

// LotsRepositorySQLC implements ports.LotsRepository using sqlc-generated queries.
type LotsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewLotsRepositorySQLC returns a lots repository backed by sqlc.
func NewLotsRepositorySQLC(queries *sqlc.Queries) *LotsRepositorySQLC {
	return &LotsRepositorySQLC{queries: queries}
}

var _ ports.LotsRepository = (*LotsRepositorySQLC)(nil)

func (r *LotsRepositorySQLC) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListLots(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to fetch lots", Handled: false}
	}
	out := make([]database.Lot, len(list))
	for i, l := range list {
		out[i] = sqlcLotToDatabase(l)
	}
	return out, nil
}

func (r *LotsRepositorySQLC) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	var list []sqlc.Lot
	var err error
	if sku != nil && *sku != "" {
		list, err = r.queries.ListLotsBySku(ctx, *sku)
	} else {
		list, err = r.queries.ListLots(ctx)
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to fetch lots", Handled: false}
	}
	out := make([]database.Lot, len(list))
	for i, l := range list {
		out[i] = sqlcLotToDatabase(l)
	}
	return out, nil
}

func (r *LotsRepositorySQLC) CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse {
	ctx := context.Background()
	var expPg pgtype.Timestamp
	if data.ExpirationDate != nil && *data.ExpirationDate != "" {
		if t, err := time.Parse("2006-01-02", *data.ExpirationDate); err == nil {
			expPg = pgtype.Timestamp{Time: t, Valid: true}
		}
	}
	status := "pending"
	if data.Status != nil && *data.Status != "" {
		status = *data.Status
	}
	qty := pgtype.Numeric{}
	_ = qty.Scan(data.Quantity)
	arg := sqlc.CreateLotParams{
		LotNumber:      data.LotNumber,
		Sku:            data.SKU,
		Quantity:       qty,
		ExpirationDate: expPg,
		Status:         status,
	}
	_, err := r.queries.CreateLot(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to create lot", Handled: false}
	}
	return nil
}

func (r *LotsRepositorySQLC) UpdateLot(id int, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	lot, err := r.queries.GetLotByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Lot not found",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	if v, ok := data["lot_number"].(string); ok {
		lot.LotNumber = v
	}
	if v, ok := data["sku"].(string); ok {
		lot.Sku = v
	}
	if v, ok := data["quantity"].(float64); ok {
		var n pgtype.Numeric
		_ = n.Scan(v)
		lot.Quantity = n
	}
	if v, ok := data["expiration_date"].(*time.Time); ok && v != nil {
		lot.ExpirationDate = pgtype.Timestamp{Time: *v, Valid: true}
	}
	if v, ok := data["status"].(string); ok {
		lot.Status = v
	}
	arg := sqlc.UpdateLotParams{
		ID:             lot.ID,
		LotNumber:      lot.LotNumber,
		Sku:            lot.Sku,
		Quantity:       lot.Quantity,
		ExpirationDate: lot.ExpirationDate,
		Status:         lot.Status,
	}
	_, err = r.queries.UpdateLot(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to update lot", Handled: false}
	}
	return nil
}

func (r *LotsRepositorySQLC) DeleteLot(id int) *responses.InternalResponse {
	ctx := context.Background()
	_, err := r.queries.GetLotByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Lot not found",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	if err := r.queries.DeleteLot(ctx, int32(id)); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to delete lot", Handled: false}
	}
	return nil
}
