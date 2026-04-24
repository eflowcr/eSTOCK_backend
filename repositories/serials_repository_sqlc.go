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

// SerialsRepositorySQLC implements ports.SerialsRepository using sqlc-generated queries.
//
// S3.5 W2-A: every method is tenant-scoped — queries enforce
// WHERE tenant_id = $N and inserts include tenant_id explicitly.
type SerialsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewSerialsRepositorySQLC returns a serials repository backed by sqlc.
func NewSerialsRepositorySQLC(queries *sqlc.Queries) *SerialsRepositorySQLC {
	return &SerialsRepositorySQLC{queries: queries}
}

var _ ports.SerialsRepository = (*SerialsRepositorySQLC)(nil)

func (r *SerialsRepositorySQLC) GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	s, err := r.queries.GetSerialByIDForTenant(ctx, sqlc.GetSerialByIDForTenantParams{ID: id, TenantID: tid})
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
	ser := serialRowToDatabase(s.ID, s.SerialNumber, s.Sku, s.Status, s.CreatedAt, s.UpdatedAt, s.TenantID)
	return &ser, nil
}

func (r *SerialsRepositorySQLC) GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	list, err := r.queries.ListSerialsBySkuForTenant(ctx, sqlc.ListSerialsBySkuForTenantParams{Sku: sku, TenantID: tid})
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las series", Handled: false}
	}
	out := make([]database.Serial, len(list))
	for i, s := range list {
		out[i] = serialRowToDatabase(s.ID, s.SerialNumber, s.Sku, s.Status, s.CreatedAt, s.UpdatedAt, s.TenantID)
	}
	return out, nil
}

func (r *SerialsRepositorySQLC) CreateSerial(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	status := "available"
	arg := sqlc.CreateSerialParams{
		SerialNumber: data.SerialNumber,
		Sku:          data.SKU,
		Status:       status,
		TenantID:     tid,
	}
	_, err = r.queries.CreateSerial(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear la serie", Handled: false}
	}
	return nil
}

func (r *SerialsRepositorySQLC) UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	s, err := r.queries.GetSerialByIDForTenant(ctx, sqlc.GetSerialByIDForTenantParams{ID: id, TenantID: tid})
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
	arg := sqlc.UpdateSerialForTenantParams{
		ID:           s.ID,
		SerialNumber: s.SerialNumber,
		Sku:          s.Sku,
		Status:       s.Status,
		TenantID:     tid,
	}
	_, err = r.queries.UpdateSerialForTenant(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar la serie", Handled: false}
	}
	return nil
}

func (r *SerialsRepositorySQLC) DeleteSerial(tenantID, id string) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	_, err = r.queries.GetSerialByIDForTenant(ctx, sqlc.GetSerialByIDForTenantParams{ID: id, TenantID: tid})
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
	if err := r.queries.DeleteSerialForTenant(ctx, sqlc.DeleteSerialForTenantParams{ID: id, TenantID: tid}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la serie", Handled: false}
	}
	return nil
}

// serialRowToDatabase converts an sqlc serials row into the GORM database model.
func serialRowToDatabase(id, serialNumber, sku, status string, createdAt, updatedAt pgtype.Timestamp, tenantID pgtype.UUID) database.Serial {
	return database.Serial{
		ID:           id,
		TenantID:     pgUUIDToString(tenantID),
		SerialNumber: serialNumber,
		SKU:          sku,
		Status:       status,
		CreatedAt:    pgTimestampToTime(createdAt),
		UpdatedAt:    pgTimestampToTime(updatedAt),
	}
}
