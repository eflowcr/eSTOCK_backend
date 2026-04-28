package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// SalesOrdersRepository implements ports.SalesOrdersRepository using GORM.
type SalesOrdersRepository struct {
	DB          *gorm.DB
	InventorySvc inventoryPickSuggestor // injected for FEFO suggestions on submit
}

// inventoryPickSuggestor is a narrow interface for FEFO pick suggestions (avoids import cycle).
type inventoryPickSuggestor interface {
	GetPickSuggestionsBySKU(sku string, qty float64) (*dto.PickSuggestionResponse, *responses.InternalResponse)
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

// nextSONumber generates "SO-YYYY-NNNN" unique per tenant per year inside tx.
// Uses SELECT MAX(...) FOR UPDATE to prevent race conditions between concurrent requests,
// mirroring the generatePONumber pattern in purchase_orders_repository.go.
func nextSONumber(tx *gorm.DB, tenantID string) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("SO-%d-", year)

	var maxNum int
	if err := tx.Raw(`
		SELECT COALESCE(MAX(
			CAST(SUBSTRING(so_number FROM LENGTH($1)+1) AS INTEGER)
		), 0)
		FROM sales_orders
		WHERE tenant_id = $2
		  AND so_number LIKE $3
		  AND deleted_at IS NULL
		FOR UPDATE
	`, prefix, tenantID, prefix+"%").Scan(&maxNum).Error; err != nil {
		return "", fmt.Errorf("generate SO number: %w", err)
	}

	return fmt.Sprintf("%s%04d", prefix, maxNum+1), nil
}

// toSalesOrderResponse builds the full API response for a header + its items.
func toSalesOrderResponse(so *database.SalesOrder, items []database.SalesOrderItem) *responses.SalesOrderResponse {
	return &responses.SalesOrderResponse{
		ID:            so.ID,
		TenantID:      so.TenantID,
		SONumber:      so.SONumber,
		CustomerID:    so.CustomerID,
		Status:        so.Status,
		ExpectedDate:  so.ExpectedDate,
		Notes:         so.Notes,
		CreatedBy:     so.CreatedBy,
		SubmittedAt:   so.SubmittedAt,
		CompletedAt:   so.CompletedAt,
		CancelledAt:   so.CancelledAt,
		PickingTaskID: so.PickingTaskID,
		CreatedAt:     so.CreatedAt,
		UpdatedAt:     so.UpdatedAt,
		Items:         items,
	}
}

// loadItems fetches all items for a sales order.
func (r *SalesOrdersRepository) loadItems(soID string) ([]database.SalesOrderItem, error) {
	var items []database.SalesOrderItem
	if err := r.DB.Where("sales_order_id = ?", soID).Find(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []database.SalesOrderItem{}
	}
	return items, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *SalesOrdersRepository) Create(tenantID, userID string, req *requests.CreateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	var result *responses.SalesOrderResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		soNumber, err := nextSONumber(tx, tenantID)
		if err != nil {
			return fmt.Errorf("generate so_number: %w", err)
		}

		id, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate so id: %w", err)
		}

		uid := userID
		so := &database.SalesOrder{
			ID:           id,
			TenantID:     tenantID,
			SONumber:     soNumber,
			CustomerID:   req.CustomerID,
			Status:       "draft",
			ExpectedDate: req.ExpectedDate,
			Notes:        req.Notes,
			CreatedBy:    &uid,
		}

		if err := tx.Create(so).Error; err != nil {
			return fmt.Errorf("create sales_order: %w", err)
		}

		items := make([]database.SalesOrderItem, 0, len(req.Items))
		for _, line := range req.Items {
			itemID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate item id: %w", err)
			}
			items = append(items, database.SalesOrderItem{
				ID:           itemID,
				SalesOrderID: id,
				ArticleSKU:   line.ArticleSKU,
				ExpectedQty:  line.ExpectedQty,
				PickedQty:    0,
				UnitPrice:    line.UnitPrice,
				Notes:        line.Notes,
			})
		}

		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("create so items: %w", err)
		}

		result = toSalesOrderResponse(so, items)
		return nil
	})

	if txErr != nil {
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al crear la orden de venta"}
	}
	return result, nil
}

func (r *SalesOrdersRepository) List(tenantID string, status, customerID, search *string, dateFrom, dateTo *string, page, limit int) (*responses.SalesOrderListResponse, *responses.InternalResponse) {
	type rawRow struct {
		database.SalesOrder
		CustomerName string `gorm:"column:customer_name"`
		ItemCount    int    `gorm:"column:item_count"`
	}

	q := r.DB.Table("sales_orders so").
		Select(`so.*, c.name AS customer_name, COUNT(soi.id) AS item_count`).
		Joins("LEFT JOIN clients c ON c.id = so.customer_id").
		Joins("LEFT JOIN sales_order_items soi ON soi.sales_order_id = so.id").
		Where("so.tenant_id = ? AND so.deleted_at IS NULL", tenantID).
		Group("so.id, c.name")

	if status != nil && *status != "" {
		q = q.Where("so.status = ?", *status)
	}
	if customerID != nil && *customerID != "" {
		q = q.Where("so.customer_id = ?", *customerID)
	}
	if search != nil && *search != "" {
		like := "%" + strings.ToLower(*search) + "%"
		q = q.Where("LOWER(so.so_number) LIKE ? OR LOWER(c.name) LIKE ?", like, like)
	}
	if dateFrom != nil && *dateFrom != "" {
		q = q.Where("so.created_at >= ?", *dateFrom)
	}
	if dateTo != nil && *dateTo != "" {
		q = q.Where("so.created_at <= ?", *dateTo)
	}

	var total int64
	countQ := r.DB.Table("sales_orders so").
		Select("COUNT(DISTINCT so.id)").
		Joins("LEFT JOIN clients c ON c.id = so.customer_id").
		Where("so.tenant_id = ? AND so.deleted_at IS NULL", tenantID)
	if status != nil && *status != "" {
		countQ = countQ.Where("so.status = ?", *status)
	}
	if customerID != nil && *customerID != "" {
		countQ = countQ.Where("so.customer_id = ?", *customerID)
	}
	if search != nil && *search != "" {
		like := "%" + strings.ToLower(*search) + "%"
		countQ = countQ.Where("LOWER(so.so_number) LIKE ? OR LOWER(c.name) LIKE ?", like, like)
	}
	if dateFrom != nil && *dateFrom != "" {
		countQ = countQ.Where("so.created_at >= ?", *dateFrom)
	}
	if dateTo != nil && *dateTo != "" {
		countQ = countQ.Where("so.created_at <= ?", *dateTo)
	}
	if err := countQ.Count(&total).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar órdenes de venta"}
	}

	offset := (page - 1) * limit
	var rows []rawRow
	if err := q.Order("so.created_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar órdenes de venta"}
	}

	items := make([]responses.SalesOrderListItem, 0, len(rows))
	for _, row := range rows {
		name := row.CustomerName
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		items = append(items, responses.SalesOrderListItem{
			ID:            row.ID,
			SONumber:      row.SONumber,
			CustomerID:    row.CustomerID,
			CustomerName:  namePtr,
			Status:        row.Status,
			ExpectedDate:  row.ExpectedDate,
			SubmittedAt:   row.SubmittedAt,
			CompletedAt:   row.CompletedAt,
			CancelledAt:   row.CancelledAt,
			PickingTaskID: row.PickingTaskID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			ItemCount:     row.ItemCount,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &responses.SalesOrderListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *SalesOrdersRepository) GetByID(id, tenantID string) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	var so database.SalesOrder
	if err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&so).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Message:    "Orden de venta no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la orden de venta"}
	}

	items, err := r.loadItems(id)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al cargar los items"}
	}

	return toSalesOrderResponse(&so, items), nil
}

func (r *SalesOrdersRepository) Update(id, tenantID string, req *requests.UpdateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	var result *responses.SalesOrderResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var so database.SalesOrder
		if err := tx.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&so).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("not_found")
			}
			return err
		}
		if so.Status != "draft" {
			return fmt.Errorf("not_draft")
		}

		if req.CustomerID != nil {
			so.CustomerID = *req.CustomerID
		}
		if req.ExpectedDate != nil {
			so.ExpectedDate = req.ExpectedDate
		}
		if req.Notes != nil {
			so.Notes = req.Notes
		}

		if err := tx.Save(&so).Error; err != nil {
			return fmt.Errorf("update so: %w", err)
		}

		if len(req.Items) > 0 {
			// Replace items: delete existing then re-insert.
			if err := tx.Where("sales_order_id = ?", id).Delete(&database.SalesOrderItem{}).Error; err != nil {
				return fmt.Errorf("delete old items: %w", err)
			}
			newItems := make([]database.SalesOrderItem, 0, len(req.Items))
			for _, line := range req.Items {
				itemID, err := tools.GenerateNanoid(tx)
				if err != nil {
					return fmt.Errorf("generate item id: %w", err)
				}
				newItems = append(newItems, database.SalesOrderItem{
					ID:           itemID,
					SalesOrderID: id,
					ArticleSKU:   line.ArticleSKU,
					ExpectedQty:  line.ExpectedQty,
					UnitPrice:    line.UnitPrice,
					Notes:        line.Notes,
				})
			}
			if err := tx.Create(&newItems).Error; err != nil {
				return fmt.Errorf("re-insert items: %w", err)
			}
		}

		items, err := r.loadItems(id)
		if err != nil {
			return fmt.Errorf("reload items: %w", err)
		}
		result = toSalesOrderResponse(&so, items)
		return nil
	})

	if txErr != nil {
		msg := txErr.Error()
		if msg == "not_found" {
			return nil, &responses.InternalResponse{
				Message:    "Orden de venta no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		if msg == "not_draft" {
			return nil, &responses.InternalResponse{
				Message:    "Solo se pueden editar órdenes en estado 'draft'",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al actualizar la orden de venta"}
	}
	return result, nil
}

func (r *SalesOrdersRepository) SoftDelete(id, tenantID string) *responses.InternalResponse {
	var so database.SalesOrder
	if err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&so).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{
				Message:    "Orden de venta no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al buscar la orden"}
	}
	if so.Status != "draft" {
		return &responses.InternalResponse{
			Message:    "Solo se pueden eliminar órdenes en estado 'draft'",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	now := time.Now()
	if err := r.DB.Model(&so).Update("deleted_at", now).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la orden"}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — Lifecycle: Submit
// ─────────────────────────────────────────────────────────────────────────────

func (r *SalesOrdersRepository) Submit(id, tenantID, userID string) (*responses.SubmitSalesOrderResult, *responses.InternalResponse) {
	var submitResult *responses.SubmitSalesOrderResult

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Load and validate SO.
		var so database.SalesOrder
		if err := tx.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&so).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("not_found")
			}
			return err
		}
		if so.Status != "draft" {
			return fmt.Errorf("not_draft")
		}

		// 2. Load items.
		var soItems []database.SalesOrderItem
		if err := tx.Where("sales_order_id = ?", id).Find(&soItems).Error; err != nil {
			return fmt.Errorf("load items: %w", err)
		}
		if len(soItems) == 0 {
			return fmt.Errorf("no_items")
		}

		// 3. For each SO item, get FEFO pick suggestions.
		type pickItem struct {
			SKU        string
			Qty        float64
			Allocs     []database.LocationAllocation
			Available  float64
		}
		pickItems := make([]pickItem, 0, len(soItems))
		var backorderCandidates []responses.BackorderCandidate

		for _, soItem := range soItems {
			var allocs []database.LocationAllocation
			available := 0.0

			if r.InventorySvc != nil {
				sugg, suggResp := r.InventorySvc.GetPickSuggestionsBySKU(soItem.ArticleSKU, soItem.ExpectedQty)
				if suggResp == nil && sugg != nil {
					allocs = sugg.Allocations
					available = sugg.TotalFound
					// If partial: cap allocs and record backorder candidate.
					if !sugg.Sufficient {
						backorderQty := soItem.ExpectedQty - sugg.TotalFound
						backorderCandidates = append(backorderCandidates, responses.BackorderCandidate{
							ArticleSKU:   soItem.ArticleSKU,
							RequestedQty: soItem.ExpectedQty,
							AvailableQty: sugg.TotalFound,
							BackorderQty: backorderQty,
						})
					}
				}
			} else {
				// No inventory service wired (tests / degraded mode): allocate with zero stock.
				available = 0
			}

			// Only include in picking task if there is available stock.
			if available > 0 {
				pickItems = append(pickItems, pickItem{
					SKU:       soItem.ArticleSKU,
					Qty:       min64(soItem.ExpectedQty, available),
					Allocs:    allocs,
					Available: available,
				})
			} else if r.InventorySvc != nil {
				// All items have no stock — still record backorder candidate if not already added.
				alreadyAdded := false
				for _, bc := range backorderCandidates {
					if bc.ArticleSKU == soItem.ArticleSKU {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					backorderCandidates = append(backorderCandidates, responses.BackorderCandidate{
						ArticleSKU:   soItem.ArticleSKU,
						RequestedQty: soItem.ExpectedQty,
						AvailableQty: 0,
						BackorderQty: soItem.ExpectedQty,
					})
				}
			}
		}

		// 4. Build picking task items JSON.
		type pickingItemJSON struct {
			SKU              string                        `json:"sku"`
			ExpectedQuantity float64                       `json:"required_qty"`
			Allocations      []database.LocationAllocation `json:"allocations"`
			Status           string                        `json:"status"`
		}
		taskItems := make([]pickingItemJSON, 0, len(pickItems))
		for _, pi := range pickItems {
			taskItems = append(taskItems, pickingItemJSON{
				SKU:              pi.SKU,
				ExpectedQuantity: pi.Qty,
				Allocations:      pi.Allocs,
				Status:           "open",
			})
		}

		itemsJSON, err := json.Marshal(taskItems)
		if err != nil {
			return fmt.Errorf("marshal picking items: %w", err)
		}

		// 5. Generate picking task ID + task_id.
		pickingID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate picking id: %w", err)
		}
		// Use SELECT nanoid(8) inside the tx for collision-free task_id generation,
		// matching the pattern used by PurchaseOrdersRepository.Submit.
		var taskID string
		if err := tx.Raw("SELECT nanoid(8)").Scan(&taskID).Error; err != nil {
			return fmt.Errorf("generate task_id: %w", err)
		}

		pickingTask := &database.PickingTask{
			ID:           pickingID,
			TaskID:       taskID,
			OrderNumber:  so.SONumber,
			CreatedBy:    userID,
			Status:       "open",
			Priority:     "normal",
			Items:        json.RawMessage(itemsJSON),
			CustomerID:   &so.CustomerID,
			TenantID:     tenantID,
			SalesOrderID: &id,
		}

		if err := tx.Create(pickingTask).Error; err != nil {
			return fmt.Errorf("create picking task: %w", err)
		}

		// 6. Advance SO status.
		now := time.Now()
		if err := tx.Exec(`
			UPDATE sales_orders
			   SET status = 'submitted', submitted_at = ?, picking_task_id = ?, updated_at = NOW()
			 WHERE id = ?`,
			now, pickingID, id,
		).Error; err != nil {
			return fmt.Errorf("update so status: %w", err)
		}

		// 7. Reload for response.
		if err := tx.Where("id = ?", id).First(&so).Error; err != nil {
			return fmt.Errorf("reload so: %w", err)
		}
		var reloadedItems []database.SalesOrderItem
		if err := tx.Where("sales_order_id = ?", id).Find(&reloadedItems).Error; err != nil {
			return fmt.Errorf("reload items: %w", err)
		}
		so.PickingTaskID = &pickingID
		so.SubmittedAt = &now
		so.Status = "submitted"

		submitResult = &responses.SubmitSalesOrderResult{
			SalesOrder:          toSalesOrderResponse(&so, reloadedItems),
			PickingTaskID:       pickingID,
			BackorderCandidates: backorderCandidates,
		}
		return nil
	})

	if txErr != nil {
		msg := txErr.Error()
		if msg == "not_found" {
			return nil, &responses.InternalResponse{
				Message:    "Orden de venta no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		if msg == "not_draft" {
			return nil, &responses.InternalResponse{
				Message:    "Solo se pueden enviar órdenes en estado 'draft'",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		if msg == "no_items" {
			return nil, &responses.InternalResponse{
				Message:    "La orden no tiene items",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al enviar la orden de venta"}
	}
	return submitResult, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — Lifecycle: Cancel
// ─────────────────────────────────────────────────────────────────────────────

func (r *SalesOrdersRepository) Cancel(id, tenantID, userID string) *responses.InternalResponse {
	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var so database.SalesOrder
		if err := tx.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&so).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("not_found")
			}
			return err
		}
		if so.Status == "completed" {
			return fmt.Errorf("already_completed")
		}
		if so.Status == "cancelled" {
			return fmt.Errorf("already_cancelled")
		}

		// If a picking task is linked and is in_progress, release reservations via status update.
		if so.PickingTaskID != nil && *so.PickingTaskID != "" {
			var pt database.PickingTask
			if err := tx.First(&pt, "id = ?", *so.PickingTaskID).Error; err == nil {
				// Cancel the picking task (B3c path — releases reservations if in_progress).
				terminal := map[string]bool{"completed": true, "completed_with_differences": true, "cancelled": true, "abandoned": true}
				if !terminal[pt.Status] {
					if err := tx.Exec(`
						UPDATE picking_tasks SET status = 'cancelled', updated_at = NOW() WHERE id = ?`,
						*so.PickingTaskID,
					).Error; err != nil {
						return fmt.Errorf("cancel picking task: %w", err)
					}
				}
			}
		}

		now := time.Now()
		if err := tx.Exec(`
			UPDATE sales_orders
			   SET status = 'cancelled', cancelled_at = ?, updated_at = NOW()
			 WHERE id = ?`, now, id,
		).Error; err != nil {
			return fmt.Errorf("cancel so: %w", err)
		}
		return nil
	})

	if txErr != nil {
		msg := txErr.Error()
		if msg == "not_found" {
			return &responses.InternalResponse{
				Message:    "Orden de venta no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		if msg == "already_completed" {
			return &responses.InternalResponse{
				Message:    "No se puede cancelar una orden completada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		if msg == "already_cancelled" {
			return &responses.InternalResponse{
				Message:    "La orden ya está cancelada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return &responses.InternalResponse{Error: txErr, Message: "Error al cancelar la orden de venta"}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3 — Picking auto-link: UpdatePickedQty
// ─────────────────────────────────────────────────────────────────────────────

// UpdatePickedQty updates sales_order_items.picked_qty and advances SO status.
// Returns the new SO status ('completed' | 'partial' | '') so CompletePickingTask can trigger DN/BO.
func (r *SalesOrdersRepository) UpdatePickedQty(salesOrderID string, pickedPerSKU map[string]float64) (string, *responses.InternalResponse) {
	var finalStatus string

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var soItems []database.SalesOrderItem
		if err := tx.Where("sales_order_id = ?", salesOrderID).Find(&soItems).Error; err != nil {
			return fmt.Errorf("load so items: %w", err)
		}

		allFulfilled := true
		anyPicked := false

		for i := range soItems {
			additional := pickedPerSKU[soItems[i].ArticleSKU]
			if additional > 0 {
				soItems[i].PickedQty += additional
				anyPicked = true
				if err := tx.Exec(`
					UPDATE sales_order_items SET picked_qty = ? WHERE id = ?`,
					soItems[i].PickedQty, soItems[i].ID,
				).Error; err != nil {
					return fmt.Errorf("update picked_qty for %s: %w", soItems[i].ArticleSKU, err)
				}
			}
			if soItems[i].PickedQty < soItems[i].ExpectedQty {
				allFulfilled = false
			}
		}

		if !anyPicked {
			return nil // nothing changed
		}

		if allFulfilled {
			finalStatus = "completed"
			if err := tx.Exec(`
				UPDATE sales_orders SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = ?`,
				salesOrderID,
			).Error; err != nil {
				return fmt.Errorf("complete so: %w", err)
			}
		} else {
			finalStatus = "partial"
			if err := tx.Exec(`
				UPDATE sales_orders SET status = 'partial', updated_at = NOW() WHERE id = ?`,
				salesOrderID,
			).Error; err != nil {
				return fmt.Errorf("partial so: %w", err)
			}
		}

		return nil
	})

	if txErr != nil {
		return "", &responses.InternalResponse{Error: txErr, Message: "Error al actualizar cantidades pickeadas"}
	}
	return finalStatus, nil
}

// min64 returns the smaller of two float64 values.
func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
