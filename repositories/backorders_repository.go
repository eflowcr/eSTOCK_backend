package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// BackordersRepository implements ports.BackordersRepository using GORM.
type BackordersRepository struct {
	DB           *gorm.DB
	InventorySvc backorderInventorySuggestor // narrow interface for FEFO pick suggestions
}

// backorderInventorySuggestor is a narrow interface to get pick suggestions for BO2.
type backorderInventorySuggestor interface {
	GetPickSuggestionsBySKU(sku string, qty float64) (*dto.PickSuggestionResponse, *responses.InternalResponse)
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func toBackorderResponse(b *database.Backorder) *responses.BackorderResponse {
	return &responses.BackorderResponse{
		ID:                     b.ID,
		TenantID:               b.TenantID,
		OriginalSalesOrderID:   b.OriginalSalesOrderID,
		ArticleSKU:             b.ArticleSKU,
		RemainingQty:           b.RemainingQty,
		Status:                 b.Status,
		GeneratedPickingTaskID: b.GeneratedPickingTaskID,
		FulfilledAt:            b.FulfilledAt,
		CreatedAt:              b.CreatedAt,
		UpdatedAt:              b.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BO1 — CreateBackorders: called from picking_task_repository (not a port method)
// ─────────────────────────────────────────────────────────────────────────────

// BackorderCreationParam describes one pending backorder line.
type BackorderCreationParam struct {
	TenantID             string
	OriginalSalesOrderID string
	ArticleSKU           string
	RemainingQty         float64
}

// CreateBackorders inserts backorder rows inside a transaction.
// Called from within the picking completion flow.
func CreateBackorders(tx *gorm.DB, params []BackorderCreationParam) error {
	for _, p := range params {
		var boID string
		if err := tx.Raw("SELECT nanoid()").Scan(&boID).Error; err != nil {
			return fmt.Errorf("generate backorder id for %s: %w", p.ArticleSKU, err)
		}
		bo := &database.Backorder{
			ID:                   boID,
			TenantID:             p.TenantID,
			OriginalSalesOrderID: p.OriginalSalesOrderID,
			ArticleSKU:           p.ArticleSKU,
			RemainingQty:         p.RemainingQty,
			Status:               "pending",
		}
		if err := tx.Create(bo).Error; err != nil {
			return fmt.Errorf("create backorder for %s: %w", p.ArticleSKU, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ports.BackordersRepository implementation
// ─────────────────────────────────────────────────────────────────────────────

// List returns paginated backorders for a tenant.
func (r *BackordersRepository) List(tenantID string, status, soID *string, page, limit int) (*responses.BackorderListResponse, *responses.InternalResponse) {
	q := r.DB.Model(&database.Backorder{}).Where("tenant_id = ?", tenantID)
	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}
	if soID != nil && *soID != "" {
		q = q.Where("original_sales_order_id = ?", *soID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar backorders"}
	}

	offset := (page - 1) * limit
	var bos []database.Backorder
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&bos).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar backorders"}
	}

	items := make([]responses.BackorderResponse, 0, len(bos))
	for i := range bos {
		items = append(items, *toBackorderResponse(&bos[i]))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &responses.BackorderListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetByID returns a single backorder scoped to tenantID.
func (r *BackordersRepository) GetByID(id, tenantID string) (*responses.BackorderResponse, *responses.InternalResponse) {
	var bo database.Backorder
	if err := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&bo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Message:    "Backorder no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener backorder"}
	}
	return toBackorderResponse(&bo), nil
}

// Fulfill creates a new picking task for a pending backorder (BO2).
// Enforces max depth=1: picking tasks sourced from backorders will not generate further backorders.
func (r *BackordersRepository) Fulfill(id, tenantID, userID string) (*responses.FulfillBackorderResult, *responses.InternalResponse) {
	var result *responses.FulfillBackorderResult

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Load and validate backorder.
		var bo database.Backorder
		if err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&bo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("not_found")
			}
			return err
		}
		if bo.Status != "pending" {
			return fmt.Errorf("not_pending")
		}

		// 2. Get FEFO pick suggestions.
		var allocs []database.LocationAllocation
		available := 0.0
		if r.InventorySvc != nil {
			sugg, suggResp := r.InventorySvc.GetPickSuggestionsBySKU(bo.ArticleSKU, bo.RemainingQty)
			if suggResp == nil && sugg != nil {
				allocs = sugg.Allocations
				available = sugg.TotalFound
			}
		}

		if available <= 0 {
			return fmt.Errorf("no_stock")
		}

		// 3. Load original SO to get SO number and customer ID.
		var so database.SalesOrder
		if err := tx.First(&so, "id = ?", bo.OriginalSalesOrderID).Error; err != nil {
			return fmt.Errorf("load so: %w", err)
		}

		// 4. Build picking task item.
		type pickingItemJSON struct {
			SKU              string                        `json:"sku"`
			ExpectedQuantity float64                       `json:"required_qty"`
			Allocations      []database.LocationAllocation `json:"allocations"`
			Status           string                        `json:"status"`
		}

		qty := bo.RemainingQty
		if available < qty {
			qty = available
		}

		taskItems := []pickingItemJSON{
			{
				SKU:              bo.ArticleSKU,
				ExpectedQuantity: qty,
				Allocations:      allocs,
				Status:           "open",
			},
		}
		itemsJSON, err := json.Marshal(taskItems)
		if err != nil {
			return fmt.Errorf("marshal picking items: %w", err)
		}

		// 5. Generate IDs.
		pickingID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate picking id: %w", err)
		}
		var taskID string
		if err := tx.Raw("SELECT nanoid(8)").Scan(&taskID).Error; err != nil {
			return fmt.Errorf("generate task_id: %w", err)
		}

		// 6. Create picking task — linked back to original SO and to this backorder.
		// source_backorder_id being set prevents further backorder generation on completion.
		soID := bo.OriginalSalesOrderID
		custID := so.CustomerID
		pickingTask := &database.PickingTask{
			ID:                pickingID,
			TaskID:            taskID,
			OrderNumber:       so.SONumber,
			CreatedBy:         userID,
			Status:            "open",
			Priority:          "high", // backorder fulfillments are high priority
			Items:             json.RawMessage(itemsJSON),
			CustomerID:        &custID,
			TenantID:          tenantID,
			SalesOrderID:      &soID,
			SourceBackorderID: &id, // max depth=1 flag
		}
		if err := tx.Create(pickingTask).Error; err != nil {
			return fmt.Errorf("create picking task: %w", err)
		}

		// 7. Update backorder with the generated picking task ID.
		now := time.Now()
		if err := tx.Exec(`
			UPDATE backorders
			   SET generated_picking_task_id = ?, updated_at = ?
			 WHERE id = ?`,
			pickingID, now, id,
		).Error; err != nil {
			return fmt.Errorf("update backorder picking task: %w", err)
		}

		// Reload for response.
		if err := tx.Where("id = ?", id).First(&bo).Error; err != nil {
			return fmt.Errorf("reload backorder: %w", err)
		}

		result = &responses.FulfillBackorderResult{
			Backorder:     toBackorderResponse(&bo),
			PickingTaskID: pickingID,
		}
		return nil
	})

	if txErr != nil {
		msg := txErr.Error()
		switch msg {
		case "not_found":
			return nil, &responses.InternalResponse{
				Message:    "Backorder no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		case "not_pending":
			return nil, &responses.InternalResponse{
				Message:    "Solo se pueden fulfilliar backorders en estado 'pending'",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		case "no_stock":
			return nil, &responses.InternalResponse{
				Message:    "No hay stock disponible para fulfilliar este backorder",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al fulfilliar backorder"}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateFulfilledBackorder: called from CompletePickingTask when source_backorder_id is set
// ─────────────────────────────────────────────────────────────────────────────

// UpdateFulfilledBackorder updates backorder remaining_qty and sets status='fulfilled' if qty <= 0.
// Called from picking_task_repository after completing a backorder-sourced picking task.
func UpdateFulfilledBackorder(db *gorm.DB, backorderID string, pickedPerSKU map[string]float64) error {
	var bo database.Backorder
	if err := db.First(&bo, "id = ?", backorderID).Error; err != nil {
		return fmt.Errorf("load backorder %s: %w", backorderID, err)
	}

	newRemaining := bo.RemainingQty - pickedPerSKU[bo.ArticleSKU]
	if newRemaining < 0 {
		newRemaining = 0
	}

	now := time.Now()
	if newRemaining <= 0 {
		return db.Exec(`
			UPDATE backorders
			   SET remaining_qty = 0, status = 'fulfilled', fulfilled_at = ?, updated_at = ?
			 WHERE id = ?`,
			now, now, backorderID,
		).Error
	}
	return db.Exec(`
		UPDATE backorders
		   SET remaining_qty = ?, updated_at = ?
		 WHERE id = ?`,
		newRemaining, now, backorderID,
	).Error
}
