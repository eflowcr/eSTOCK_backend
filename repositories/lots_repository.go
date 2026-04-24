package repositories

import (
	"errors"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type LotsRepository struct {
	DB *gorm.DB
}

// GetAllLots returns lots for a single tenant ordered by creation date (newest first).
// S3.5 W2-B: tenant filter is mandatory; an empty tenantID yields zero rows by design.
func (r *LotsRepository) GetAllLots(tenantID string) ([]database.Lot, *responses.InternalResponse) {
	var lots []database.Lot

	err := r.DB.Table(database.Lot{}.TableName()).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&lots).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots",
			Handled: false,
		}
	}

	return lots, nil
}

// GetLotsBySKU returns the tenant's lots, optionally filtered by SKU.
// S3.5 W2-B: tenant filter is always applied; SKU is an additional narrowing filter.
func (r *LotsRepository) GetLotsBySKU(tenantID string, sku *string) ([]database.Lot, *responses.InternalResponse) {
	var lots []database.Lot

	query := r.DB.Table(database.Lot{}.TableName()).Where("tenant_id = ?", tenantID)

	if sku != nil && *sku != "" {
		query = query.Where("sku = ?", *sku)
	} else {
		query = query.Order("created_at DESC")
	}

	err := query.Find(&lots).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots",
			Handled: false,
		}
	}

	return lots, nil
}

// CreateLot inserts a new lot owned by tenantID. S3.5 W2-B.
func (r *LotsRepository) CreateLot(tenantID string, data *requests.CreateLotRequest) *responses.InternalResponse {
	now := tools.GetCurrentTime()

	// Parse string to time.Time
	var expirationDate time.Time

	if data.ExpirationDate != nil {
		expirationDate, _ = time.Parse("2006-01-02", *data.ExpirationDate)
	}

	lotID, idErr := tools.GenerateNanoid(r.DB)
	if idErr != nil {
		return &responses.InternalResponse{
			Error:   idErr,
			Message: "Failed to generate lot id",
			Handled: false,
		}
	}

	lot := &database.Lot{
		ID:             lotID,
		TenantID:       tenantID,
		LotNumber:      data.LotNumber,
		SKU:            data.SKU,
		Quantity:       data.Quantity,
		ExpirationDate: &expirationDate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := r.DB.Create(lot).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create lot",
			Handled: false,
		}
	}

	return nil
}

// UpdateLot mutates a lot scoped to the calling tenant. S3.5 W2-B: cross-tenant updates
// return NotFound to avoid leaking the existence of a row owned by another tenant.
func (r *LotsRepository) UpdateLot(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	var lot database.Lot

	err := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&lot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Message:    "Lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to retrieve lot",
			Handled: false,
		}
	}

	protectedFields := map[string]bool{
		"id":         true,
		"tenant_id":  true,
		"created_at": true,
	}

	for k := range protectedFields {
		delete(data, k)
	}

	data["updated_at"] = tools.GetCurrentTime()

	if err := r.DB.Table(database.Lot{}.TableName()).Where(
		"id = ? AND tenant_id = ?", id, tenantID,
	).Updates(data).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update lot",
			Handled: false,
		}
	}

	return nil
}

// DeleteLot removes a lot scoped to the calling tenant. S3.5 W2-B.
func (r *LotsRepository) DeleteLot(tenantID, id string) *responses.InternalResponse {
	result := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&database.Lot{})
	if result.Error != nil {
		return &responses.InternalResponse{
			Error:   result.Error,
			Message: "Failed to delete lot",
			Handled: false,
		}
	}

	if result.RowsAffected == 0 {
		return &responses.InternalResponse{
			Message:    "Lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}

	return nil
}

// GetLotByID is the internal-use lookup (no tenant filter). Used by GetLotTrace itself
// after the tenant guard has already run, and by intra-domain joins where the parent
// row has already been tenant-checked.
func (r *LotsRepository) GetLotByID(id string) (*database.Lot, *responses.InternalResponse) {
	var lot database.Lot
	err := r.DB.First(&lot, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Message:    "Lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	return &lot, nil
}

// GetLotByIDForTenant scopes the lookup to tenantID. S3.5 W2-B: HTTP callers must use
// this variant so cross-tenant lot enumeration is impossible.
func (r *LotsRepository) GetLotByIDForTenant(id, tenantID string) (*database.Lot, *responses.InternalResponse) {
	var lot database.Lot
	err := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&lot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Message:    "Lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to retrieve lot", Handled: false}
	}
	return &lot, nil
}

// GetLotTrace returns the full provenance trace for a lot owned by tenantID.
// S3.5 W2-B: lot lookup is tenant-scoped; downstream joins inherit isolation
// via inventory_movements/inventory_lots which were tenant-scoped in S2.5.
func (r *LotsRepository) GetLotTrace(tenantID, lotID string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	// 1. Lot data (tenant-scoped — guards trace endpoint from cross-tenant enumeration).
	lot, resp := r.GetLotByIDForTenant(lotID, tenantID)
	if resp != nil {
		return nil, resp
	}

	trace := &responses.LotTraceResponse{
		Lot: responses.LotTraceLot{
			ID:             lot.ID,
			LotNumber:      lot.LotNumber,
			SKU:            lot.SKU,
			ExpirationDate: lot.ExpirationDate,
			ManufacturedAt: lot.ManufacturedAt,
			BestBeforeDate: lot.BestBeforeDate,
			Status:         lot.Status,
		},
	}

	// 2. Movements
	type movRow struct {
		ID            string
		MovementType  string
		Quantity      float64
		BeforeQty     *float64
		AfterQty      *float64
		Location      string
		ReferenceType *string
		ReferenceID   *string
		UserID        *string
		UnitCost      *float64
		CreatedAt     time.Time
	}
	var movRows []movRow
	r.DB.Raw(`SELECT id, movement_type, quantity, before_qty, after_qty, location,
	               reference_type, reference_id, user_id, unit_cost, created_at
	          FROM inventory_movements WHERE lot_id = ? ORDER BY created_at ASC`, lotID).Scan(&movRows)

	trace.Movements = make([]responses.LotTraceMovement, len(movRows))
	for i, m := range movRows {
		trace.Movements[i] = responses.LotTraceMovement{
			ID:            m.ID,
			Type:          m.MovementType,
			Qty:           m.Quantity,
			BeforeQty:     m.BeforeQty,
			AfterQty:      m.AfterQty,
			Location:      m.Location,
			ReferenceType: m.ReferenceType,
			ReferenceID:   m.ReferenceID,
			UserID:        m.UserID,
			UnitCost:      m.UnitCost,
			CreatedAt:     m.CreatedAt,
		}
	}

	// 3. Origin: first INBOUND movement with reference_type = 'receiving_task'
	for _, m := range movRows {
		if m.ReferenceType != nil && *m.ReferenceType == "receiving_task" &&
			(m.MovementType == "inbound" || m.MovementType == "INBOUND") {
			origin := &responses.LotTraceOrigin{
				ReceivingTaskID: *m.ReferenceID,
				ReceivedAt:      m.CreatedAt,
			}
			// Fetch supplier from the receiving task
			type rtRow struct {
				SupplierID   *string
				SupplierCode *string
				SupplierName *string
			}
			var rt rtRow
			r.DB.Raw(`SELECT rt.supplier_id, c.code AS supplier_code, c.name AS supplier_name
				FROM receiving_tasks rt
				LEFT JOIN clients c ON c.id = rt.supplier_id
				WHERE rt.id = ?`, *m.ReferenceID).Scan(&rt)
			if rt.SupplierID != nil {
				origin.Supplier = &responses.LotTraceSupplier{
					ID:   *rt.SupplierID,
					Code: derefStr(rt.SupplierCode),
					Name: derefStr(rt.SupplierName),
				}
			}
			trace.Origin = origin
			break
		}
	}

	// 4. Current stock from inventory_lots
	type stockRow struct {
		Location string
		Qty      float64
	}
	var stockRows []stockRow
	r.DB.Raw(`SELECT location, SUM(quantity) AS qty FROM inventory_lots WHERE lot_id = ? GROUP BY location`, lotID).Scan(&stockRows)

	byLoc := make([]responses.LotTraceLocationQty, len(stockRows))
	var total float64
	for i, s := range stockRows {
		byLoc[i] = responses.LotTraceLocationQty{Location: s.Location, Qty: s.Qty}
		total += s.Qty
	}
	trace.CurrentStock = responses.LotTraceCurrentStock{TotalQty: total, ByLocation: byLoc}

	return trace, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
