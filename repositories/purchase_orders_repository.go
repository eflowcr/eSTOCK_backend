package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// PurchaseOrdersRepository implements ports.PurchaseOrdersRepository using GORM.
// Consistent with ReceivingTasksRepository (GORM-based, raw SQL where needed).
type PurchaseOrdersRepository struct {
	DB *gorm.DB
}

var _ ports.PurchaseOrdersRepository = (*PurchaseOrdersRepository)(nil)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// generatePONumber creates the next sequential PO number for the tenant+year inside a tx.
// Format: PO-YYYY-NNNN (zero-padded to 4 digits). Uses SELECT FOR UPDATE on MAX to prevent
// race conditions between concurrent requests.
func generatePONumber(tx *gorm.DB, tenantID string) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("PO-%d-", year)

	var maxNum int
	err := tx.Raw(`
		SELECT COALESCE(MAX(
			CAST(SUBSTRING(po_number FROM LENGTH($1)+1) AS INTEGER)
		), 0)
		FROM purchase_orders
		WHERE tenant_id = $2
		  AND po_number LIKE $3
		  AND deleted_at IS NULL
		FOR UPDATE
	`, prefix, tenantID, prefix+"%").Scan(&maxNum).Error
	if err != nil {
		return "", fmt.Errorf("generate PO number: %w", err)
	}

	return fmt.Sprintf("%s%04d", prefix, maxNum+1), nil
}

// poToView converts a database.PurchaseOrder + items slice to a response view.
func poToView(po database.PurchaseOrder, items []database.PurchaseOrderItem) responses.PurchaseOrderView {
	v := responses.PurchaseOrderView{
		ID:              po.ID,
		PONumber:        po.PONumber,
		SupplierID:      po.SupplierID,
		Status:          po.Status,
		ExpectedDate:    po.ExpectedDate,
		Notes:           po.Notes,
		CreatedBy:       po.CreatedBy,
		SubmittedAt:     po.SubmittedAt,
		CompletedAt:     po.CompletedAt,
		CancelledAt:     po.CancelledAt,
		ReceivingTaskID: po.ReceivingTaskID,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
		TenantID:        po.TenantID,
	}
	for _, it := range items {
		v.Items = append(v.Items, responses.PurchaseOrderItemView{
			ID:          it.ID,
			ArticleSKU:  it.ArticleSKU,
			ExpectedQty: it.ExpectedQty,
			ReceivedQty: it.ReceivedQty,
			RejectedQty: it.RejectedQty,
			Discrepancy: it.Discrepancy,
			UnitCost:    it.UnitCost,
			Notes:       it.Notes,
		})
	}
	return v
}

// loadItems fetches purchase_order_items for a given PO id.
func (r *PurchaseOrdersRepository) loadItems(poID string) ([]database.PurchaseOrderItem, error) {
	var items []database.PurchaseOrderItem
	if err := r.DB.Where("purchase_order_id = ?", poID).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) Create(tenantID, createdBy string, req *requests.CreatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	var result *responses.PurchaseOrderView

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Auto-generate PO number (row-locked per tenant+year).
		poNumber, err := generatePONumber(tx, tenantID)
		if err != nil {
			return err
		}

		poID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate PO id: %w", err)
		}

		now := tools.GetCurrentTime()
		po := database.PurchaseOrder{
			ID:           poID,
			TenantID:     tenantID,
			PONumber:     poNumber,
			SupplierID:   req.SupplierID,
			Status:       "draft",
			ExpectedDate: req.ExpectedDate,
			Notes:        req.Notes,
			CreatedBy:    &createdBy,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := tx.Create(&po).Error; err != nil {
			return fmt.Errorf("create purchase_order: %w", err)
		}

		// Insert items.
		items := make([]database.PurchaseOrderItem, 0, len(req.Items))
		for _, it := range req.Items {
			itemID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate PO item id: %w", err)
			}
			item := database.PurchaseOrderItem{
				ID:              itemID,
				PurchaseOrderID: poID,
				ArticleSKU:      it.ArticleSKU,
				ExpectedQty:     it.ExpectedQty,
				ReceivedQty:     0,
				RejectedQty:     0,
				UnitCost:        it.UnitCost,
				Notes:           it.Notes,
				CreatedAt:       now,
			}
			if err := tx.Create(&item).Error; err != nil {
				return fmt.Errorf("create PO item for SKU %s: %w", it.ArticleSKU, err)
			}
			items = append(items, item)
		}

		v := poToView(po, items)
		result = &v
		return nil
	})

	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al crear la orden de compra"}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GetByID
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) GetByID(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	var po database.PurchaseOrder
	if err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{Message: "Orden de compra no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la orden de compra"}
	}

	items, err := r.loadItems(id)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener los items de la orden de compra"}
	}

	v := poToView(po, items)
	return &v, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// List
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) List(tenantID string, status, supplierID, search *string, from, to *string, limit, offset int) ([]responses.PurchaseOrderView, *responses.InternalResponse) {
	query := r.DB.Model(&database.PurchaseOrder{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if supplierID != nil && *supplierID != "" {
		query = query.Where("supplier_id = ?", *supplierID)
	}
	if search != nil && *search != "" {
		pattern := "%" + *search + "%"
		query = query.Where("po_number ILIKE ?", pattern)
	}
	if from != nil && *from != "" {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil && *to != "" {
		query = query.Where("created_at <= ?", *to)
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var pos []database.PurchaseOrder
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&pos).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar órdenes de compra"}
	}

	views := make([]responses.PurchaseOrderView, 0, len(pos))
	for _, po := range pos {
		items, err := r.loadItems(po.ID)
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al cargar items de la orden de compra"}
		}
		views = append(views, poToView(po, items))
	}
	return views, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Update (draft only)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) Update(id, tenantID string, req *requests.UpdatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	var po database.PurchaseOrder
	if err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{Message: "Orden de compra no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la orden de compra"}
	}

	if po.Status != "draft" {
		return nil, &responses.InternalResponse{
			Message:    fmt.Sprintf("Solo se pueden editar órdenes de compra en estado 'draft' (actual: %s)", po.Status),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	updates := map[string]interface{}{
		"updated_at": tools.GetCurrentTime(),
	}
	if req.ExpectedDate != nil {
		updates["expected_date"] = req.ExpectedDate
	}
	if req.Notes != nil {
		updates["notes"] = req.Notes
	}

	if err := r.DB.Model(&po).Updates(updates).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar la orden de compra"}
	}

	items, err := r.loadItems(id)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener los items actualizados"}
	}

	v := poToView(po, items)
	return &v, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SoftDelete
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) SoftDelete(id, tenantID string) *responses.InternalResponse {
	var po database.PurchaseOrder
	if err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{Message: "Orden de compra no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al obtener la orden de compra"}
	}

	now := tools.GetCurrentTime()
	if err := r.DB.Model(&po).Updates(map[string]interface{}{
		"deleted_at": now,
		"updated_at": now,
	}).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la orden de compra"}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Submit — draft → submitted, auto-create receiving task
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) Submit(id, tenantID, userID string) (*responses.PurchaseOrderView, string, *responses.InternalResponse) {
	var resultView *responses.PurchaseOrderView
	var newReceivingTaskID string
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var po database.PurchaseOrder
		if err := tx.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&po).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Orden de compra no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
				return nil
			}
			return fmt.Errorf("load PO: %w", err)
		}

		if po.Status != "draft" {
			*handledResp = responses.InternalResponse{
				Message:    fmt.Sprintf("Solo se pueden someter órdenes en estado 'draft' (actual: %s)", po.Status),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
			return nil
		}

		// Load PO items to build receiving task items.
		var items []database.PurchaseOrderItem
		if err := tx.Where("purchase_order_id = ?", id).Find(&items).Error; err != nil {
			return fmt.Errorf("load PO items: %w", err)
		}
		if len(items) == 0 {
			*handledResp = responses.InternalResponse{Message: "La orden de compra no tiene items", Handled: true, StatusCode: responses.StatusBadRequest}
			return nil
		}

		// Build receiving_task items (JSONB). Use the same ReceivingTaskItemRequest shape.
		type rtItem struct {
			SKU              string  `json:"sku"`
			ExpectedQuantity int     `json:"expected_qty"`
			Location         string  `json:"location"`
			Status           *string `json:"status,omitempty"`
		}
		rtItems := make([]rtItem, 0, len(items))
		pending := "pending"
		for _, it := range items {
			rtItems = append(rtItems, rtItem{
				SKU:              it.ArticleSKU,
				ExpectedQuantity: int(it.ExpectedQty),
				Location:         "",
				Status:           &pending,
			})
		}
		itemsJSON, err := json.Marshal(rtItems)
		if err != nil {
			return fmt.Errorf("marshal receiving task items: %w", err)
		}

		// Generate receiving task IDs.
		rtID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate receiving task id: %w", err)
		}

		// Generate a task_id (short human-readable).
		var taskID string
		if err := tx.Raw("SELECT nanoid(8)").Scan(&taskID).Error; err != nil {
			return fmt.Errorf("generate task_id: %w", err)
		}

		// Generate inbound_number from PO number.
		inboundNumber := "REC-" + po.PONumber

		now := tools.GetCurrentTime()

		// Create receiving_task with purchase_order_id link.
		rt := database.ReceivingTask{
			ID:              rtID,
			TaskID:          taskID,
			InboundNumber:   inboundNumber,
			CreatedBy:       userID,
			Status:          "open",
			Priority:        "normal",
			Items:           itemsJSON,
			SupplierID:      &po.SupplierID,
			TenantID:        tenantID,
			PurchaseOrderID: &id,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := tx.Create(&rt).Error; err != nil {
			return fmt.Errorf("create receiving task: %w", err)
		}

		// Update PO: status=submitted, submitted_at, receiving_task_id.
		submittedAt := now
		updates := map[string]interface{}{
			"status":            "submitted",
			"submitted_at":      submittedAt,
			"receiving_task_id": rtID,
			"updated_at":        now,
		}
		if err := tx.Model(&po).Updates(updates).Error; err != nil {
			return fmt.Errorf("update PO status: %w", err)
		}

		po.Status = "submitted"
		po.SubmittedAt = &submittedAt
		po.ReceivingTaskID = &rtID
		po.UpdatedAt = now

		v := poToView(po, items)
		resultView = &v
		newReceivingTaskID = rtID
		return nil
	})

	if err != nil {
		return nil, "", &responses.InternalResponse{Error: err, Message: "Error al someter la orden de compra"}
	}
	if handledResp.Handled || handledResp.Error != nil {
		return nil, "", handledResp
	}
	return resultView, newReceivingTaskID, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cancel
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) Cancel(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	var resultView *responses.PurchaseOrderView
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var po database.PurchaseOrder
		if err := tx.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&po).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Orden de compra no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
				return nil
			}
			return fmt.Errorf("load PO: %w", err)
		}

		if po.Status == "completed" {
			*handledResp = responses.InternalResponse{
				Message:    "No se puede cancelar una orden de compra ya completada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
			return nil
		}
		if po.Status == "cancelled" {
			*handledResp = responses.InternalResponse{
				Message:    "La orden de compra ya está cancelada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
			return nil
		}

		now := tools.GetCurrentTime()
		updates := map[string]interface{}{
			"status":       "cancelled",
			"cancelled_at": now,
			"updated_at":   now,
		}
		if err := tx.Model(&po).Updates(updates).Error; err != nil {
			return fmt.Errorf("cancel PO: %w", err)
		}

		po.Status = "cancelled"
		po.CancelledAt = &now
		po.UpdatedAt = now

		var items []database.PurchaseOrderItem
		if err := tx.Where("purchase_order_id = ?", id).Find(&items).Error; err != nil {
			return fmt.Errorf("load items: %w", err)
		}

		v := poToView(po, items)
		resultView = &v
		return nil
	})

	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al cancelar la orden de compra"}
	}
	if handledResp.Handled || handledResp.Error != nil {
		return nil, handledResp
	}
	return resultView, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateReceivedQty — called by receiving completion flow (PO3)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PurchaseOrdersRepository) UpdateReceivedQty(purchaseOrderID string, itemUpdates []database.PurchaseOrderItemQtyUpdate) *responses.InternalResponse {
	if len(itemUpdates) == 0 {
		return nil
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Update received_qty / rejected_qty per item by sku.
		for _, upd := range itemUpdates {
			res := tx.Model(&database.PurchaseOrderItem{}).
				Where("purchase_order_id = ? AND article_sku = ?", purchaseOrderID, upd.ArticleSKU).
				Updates(map[string]interface{}{
					"received_qty": gorm.Expr("received_qty + ?", upd.ReceivedQty),
					"rejected_qty": gorm.Expr("rejected_qty + ?", upd.RejectedQty),
				})
			if res.Error != nil {
				return fmt.Errorf("update qty for sku %s: %w", upd.ArticleSKU, res.Error)
			}
		}

		// Re-read all items to compute PO completion state.
		var items []database.PurchaseOrderItem
		if err := tx.Where("purchase_order_id = ?", purchaseOrderID).Find(&items).Error; err != nil {
			return fmt.Errorf("reload PO items: %w", err)
		}

		allFulfilled := true
		anyFulfilled := false
		for _, it := range items {
			fulfilled := it.ExpectedQty <= (it.ReceivedQty + it.RejectedQty)
			if fulfilled {
				anyFulfilled = true
			} else {
				allFulfilled = false
			}
		}

		var newStatus string
		now := tools.GetCurrentTime()
		updates := map[string]interface{}{"updated_at": now}

		if allFulfilled {
			newStatus = "completed"
			updates["status"] = newStatus
			updates["completed_at"] = now
		} else if anyFulfilled {
			newStatus = "partial"
			updates["status"] = newStatus
		} else {
			// Nothing fulfilled yet — no status change needed.
			return nil
		}

		if err := tx.Model(&database.PurchaseOrder{}).
			Where("id = ?", purchaseOrderID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update PO status to %s: %w", newStatus, err)
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar cantidades recibidas en la orden de compra"}
	}
	return nil
}
