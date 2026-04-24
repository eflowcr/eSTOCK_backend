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

// StockSettingsRepositorySQLC implements ports.StockSettingsRepository using sqlc-generated queries.
type StockSettingsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewStockSettingsRepositorySQLC returns a stock_settings repository backed by sqlc.
func NewStockSettingsRepositorySQLC(queries *sqlc.Queries) *StockSettingsRepositorySQLC {
	return &StockSettingsRepositorySQLC{queries: queries}
}

var _ ports.StockSettingsRepository = (*StockSettingsRepositorySQLC)(nil)

func (r *StockSettingsRepositorySQLC) GetOrCreate(tenantID string) (*database.StockSetting, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	s, err := r.queries.GetStockSettings(ctx, tid)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la configuración", Handled: false}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		// Create with defaults — INSERT ON CONFLICT DO NOTHING pattern via UpsertStockSettings
		defaults := sqlc.UpsertStockSettingsParams{
			TenantID:                  tid,
			ValuationMethod:           "avco",
			PickBatchBasedOn:          "fefo",
			OverReceiptAllowancePct:   pgtype.Numeric{},
			OverDeliveryAllowancePct:  pgtype.Numeric{},
			OverPickingAllowancePct:   pgtype.Numeric{},
			AutoReserveStock:          true,
			AllowPartialReservation:   true,
			ExpiryAlertDays:           30,
			AutoCreateMaterialRequest: false,
			PartialDeliveryPolicy:     "immediate",
		}
		s, err = r.queries.UpsertStockSettings(ctx, defaults)
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al crear configuración por defecto", Handled: false}
		}
	}

	result := sqlcStockSettingToDatabase(s)
	return &result, nil
}

func (r *StockSettingsRepositorySQLC) Upsert(tenantID string, data *requests.UpdateStockSettingsRequest) (*database.StockSetting, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	arg := sqlc.UpsertStockSettingsParams{
		TenantID:                  tid,
		ValuationMethod:           data.ValuationMethod,
		PickBatchBasedOn:          data.PickBatchBasedOn,
		OverReceiptAllowancePct:   floatToPgNumeric(data.OverReceiptAllowancePct),
		OverDeliveryAllowancePct:  floatToPgNumeric(data.OverDeliveryAllowancePct),
		OverPickingAllowancePct:   floatToPgNumeric(data.OverPickingAllowancePct),
		AutoReserveStock:          data.AutoReserveStock,
		AllowPartialReservation:   data.AllowPartialReservation,
		ExpiryAlertDays:           int32(data.ExpiryAlertDays),
		AutoCreateMaterialRequest: data.AutoCreateMaterialRequest,
		PartialDeliveryPolicy:     data.PartialDeliveryPolicy,
	}
	s, err := r.queries.UpsertStockSettings(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al guardar la configuración", Handled: false}
	}
	result := sqlcStockSettingToDatabase(s)
	return &result, nil
}

func sqlcStockSettingToDatabase(s sqlc.StockSetting) database.StockSetting {
	return database.StockSetting{
		TenantID:                  pgUUIDToString(s.TenantID),
		ValuationMethod:           s.ValuationMethod,
		PickBatchBasedOn:          s.PickBatchBasedOn,
		OverReceiptAllowancePct:   pgNumericToFloat(s.OverReceiptAllowancePct),
		OverDeliveryAllowancePct:  pgNumericToFloat(s.OverDeliveryAllowancePct),
		OverPickingAllowancePct:   pgNumericToFloat(s.OverPickingAllowancePct),
		AutoReserveStock:          s.AutoReserveStock,
		AllowPartialReservation:   s.AllowPartialReservation,
		ExpiryAlertDays:           int(s.ExpiryAlertDays),
		AutoCreateMaterialRequest: s.AutoCreateMaterialRequest,
		PartialDeliveryPolicy:     s.PartialDeliveryPolicy,
		UpdatedAt:                 s.UpdatedAt,
	}
}

