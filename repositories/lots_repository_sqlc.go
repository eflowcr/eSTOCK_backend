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
	"gorm.io/gorm"
)

// LotsRepositorySQLC implements ports.LotsRepository using sqlc-generated queries.
// DB is optional; used by GetLotTrace (which requires cross-table raw SQL not modeled in sqlc).
type LotsRepositorySQLC struct {
	queries *sqlc.Queries
	DB      *gorm.DB
}

// NewLotsRepositorySQLC returns a lots repository backed by sqlc.
func NewLotsRepositorySQLC(queries *sqlc.Queries) *LotsRepositorySQLC {
	return &LotsRepositorySQLC{queries: queries}
}

// NewLotsRepositorySQLCWithGORM returns a lots repository backed by sqlc with GORM for trace queries.
func NewLotsRepositorySQLCWithGORM(queries *sqlc.Queries, db *gorm.DB) *LotsRepositorySQLC {
	return &LotsRepositorySQLC{queries: queries, DB: db}
}

var _ ports.LotsRepository = (*LotsRepositorySQLC)(nil)

// tenantBadRequest converts a tenant_id parse error into a 400 response.
func tenantBadRequest(err error) *responses.InternalResponse {
	return &responses.InternalResponse{
		Error:      err,
		Message:    "tenant_id inválido",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
}

func (r *LotsRepositorySQLC) GetAllLots(tenantID string) ([]database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, tenantBadRequest(err)
	}
	list, err := r.queries.ListLots(ctx, tid)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to fetch lots", Handled: false}
	}
	out := make([]database.Lot, len(list))
	for i, l := range list {
		out[i] = sqlcLotToDatabase(l)
	}
	return out, nil
}

func (r *LotsRepositorySQLC) GetLotsBySKU(tenantID string, sku *string) ([]database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, tenantBadRequest(err)
	}
	var list []sqlc.Lot
	if sku != nil && *sku != "" {
		list, err = r.queries.ListLotsBySkuForTenant(ctx, sqlc.ListLotsBySkuForTenantParams{TenantID: tid, Sku: *sku})
	} else {
		list, err = r.queries.ListLots(ctx, tid)
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

func (r *LotsRepositorySQLC) CreateLot(tenantID string, data *requests.CreateLotRequest) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return tenantBadRequest(err)
	}
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
		TenantID:       tid,
		LotNumber:      data.LotNumber,
		Sku:            data.SKU,
		Quantity:       qty,
		ExpirationDate: expPg,
		Status:         status,
		LotNotes:       ptrStringToPgText(data.LotNotes),
		ManufacturedAt: ptrStringToPgDate(data.ManufacturedAt),
		BestBeforeDate: ptrStringToPgDate(data.BestBeforeDate),
	}
	_, err = r.queries.CreateLot(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to create lot", Handled: false}
	}
	return nil
}

func (r *LotsRepositorySQLC) UpdateLot(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return tenantBadRequest(err)
	}
	// Tenant-scoped fetch — cross-tenant lookup returns NotFound, never the actual row.
	lot, err := r.queries.GetLotByIDForTenant(ctx, sqlc.GetLotByIDForTenantParams{ID: id, TenantID: tid})
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
		TenantID:       tid,
		LotNumber:      lot.LotNumber,
		Sku:            lot.Sku,
		Quantity:       lot.Quantity,
		ExpirationDate: lot.ExpirationDate,
		Status:         lot.Status,
		LotNotes:       lot.LotNotes,
		ManufacturedAt: lot.ManufacturedAt,
		BestBeforeDate: lot.BestBeforeDate,
	}
	_, err = r.queries.UpdateLot(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to update lot", Handled: false}
	}
	return nil
}

func (r *LotsRepositorySQLC) DeleteLot(tenantID, id string) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return tenantBadRequest(err)
	}
	// Tenant-scoped existence check — guarantees Delete cannot affect another tenant's row.
	if _, err := r.queries.GetLotByIDForTenant(ctx, sqlc.GetLotByIDForTenantParams{ID: id, TenantID: tid}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{
				Message:    "Lot not found",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	if err := r.queries.DeleteLot(ctx, sqlc.DeleteLotParams{ID: id, TenantID: tid}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to delete lot", Handled: false}
	}
	return nil
}

// GetLotByID is the internal-use lookup (no tenant filter). HTTP callers must use
// GetLotByIDForTenant instead.
func (r *LotsRepositorySQLC) GetLotByID(id string) (*database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	l, err := r.queries.GetLotByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Lot not found",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	lot := sqlcLotToDatabase(l)
	return &lot, nil
}

// GetLotByIDForTenant scopes the lookup to tenantID. S3.5 W2-B.
func (r *LotsRepositorySQLC) GetLotByIDForTenant(id, tenantID string) (*database.Lot, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, tenantBadRequest(err)
	}
	l, err := r.queries.GetLotByIDForTenant(ctx, sqlc.GetLotByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Lot not found",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	lot := sqlcLotToDatabase(l)
	return &lot, nil
}

// GetLotTrace delegates to the GORM-based LotsRepository when DB is available.
// S3.5 W2-B: tenantID propagates so the lot lookup itself is tenant-scoped.
func (r *LotsRepositorySQLC) GetLotTrace(tenantID, lotID string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	if r.DB == nil {
		return nil, &responses.InternalResponse{
			Message:    "GetLotTrace requiere conexión GORM — configure DB en el repositorio SQLC",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		}
	}
	gormRepo := &LotsRepository{DB: r.DB}
	return gormRepo.GetLotTrace(tenantID, lotID)
}
