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
)

// SerialsRepositorySQLC implements ports.SerialsRepository using sqlc-generated queries.
type SerialsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewSerialsRepositorySQLC returns a serials repository backed by sqlc.
func NewSerialsRepositorySQLC(queries *sqlc.Queries) *SerialsRepositorySQLC {
	return &SerialsRepositorySQLC{queries: queries}
}

var _ ports.SerialsRepository = (*SerialsRepositorySQLC)(nil)

func (r *SerialsRepositorySQLC) GetSerialByID(id int) (*database.Serial, *responses.InternalResponse) {
	ctx := context.Background()
	s, err := r.queries.GetSerialByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Serie no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la serie", Handled: false}
	}
	ser := sqlcSerialToDatabase(s)
	return &ser, nil
}

func (r *SerialsRepositorySQLC) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListSerialsBySku(ctx, sku)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las series", Handled: false}
	}
	out := make([]database.Serial, len(list))
	for i, s := range list {
		out[i] = sqlcSerialToDatabase(s)
	}
	return out, nil
}

func (r *SerialsRepositorySQLC) CreateSerial(data *requests.CreateSerialRequest) *responses.InternalResponse {
	ctx := context.Background()
	status := "available"
	arg := sqlc.CreateSerialParams{
		SerialNumber: data.SerialNumber,
		Sku:          data.SKU,
		Status:       status,
	}
	_, err := r.queries.CreateSerial(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear la serie", Handled: false}
	}
	return nil
}

func (r *SerialsRepositorySQLC) UpdateSerial(id int, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	s, err := r.queries.GetSerialByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Serie no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al obtener la serie", Handled: false}
	}
	if v, ok := data["serial_number"].(string); ok {
		s.SerialNumber = v
	}
	if v, ok := data["sku"].(string); ok {
		s.Sku = v
	}
	if v, ok := data["status"].(string); ok {
		s.Status = v
	}
	arg := sqlc.UpdateSerialParams{
		ID:           s.ID,
		SerialNumber: s.SerialNumber,
		Sku:          s.Sku,
		Status:       s.Status,
	}
	_, err = r.queries.UpdateSerial(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar la serie", Handled: false}
	}
	return nil
}

func (r *SerialsRepositorySQLC) DeleteSerial(id int) *responses.InternalResponse {
	ctx := context.Background()
	_, err := r.queries.GetSerialByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Serie no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al obtener la serie", Handled: false}
	}
	if err := r.queries.DeleteSerial(ctx, int32(id)); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la serie", Handled: false}
	}
	return nil
}
