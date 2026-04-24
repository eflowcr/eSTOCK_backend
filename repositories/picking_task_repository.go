package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type PickingTaskRepository struct {
	DB               *gorm.DB
	AuditService     *services.AuditService     // injected via wire for audit logging
	NotificationsSvc *services.NotificationsService // optional: emit task events
}

// validPickingTransitions declares the allowed status transitions.
// Terminal states (completed, completed_with_differences, cancelled, abandoned)
// have no outgoing transitions.
var validPickingTransitions = map[string]map[string]bool{
	"open":        {"assigned": true, "in_progress": true, "cancelled": true, "abandoned": true},
	"assigned":    {"open": true, "in_progress": true, "cancelled": true, "abandoned": true},
	"in_progress": {"completed": true, "completed_with_differences": true, "cancelled": true, "abandoned": true},
}

// isValidPickingTransition returns true when the status change is permitted.
// Same-state (no-op) is always true. Terminal states have no outgoing transitions → false.
func isValidPickingTransition(current, next string) bool {
	if current == next {
		return true
	}
	if allowed, ok := validPickingTransitions[current]; ok {
		return allowed[next]
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// H2 — parsePickingItemsWithLegacyFallback
// ─────────────────────────────────────────────────────────────────────────────

// parsePickingItemsWithLegacyFallback accepts both the new format (items with
// allocations) and the old format (item with a single "location" string but no
// allocations). Legacy items get a synthetic single-allocation so the rest of
// the code can treat all items uniformly.
func parsePickingItemsWithLegacyFallback(raw json.RawMessage) ([]requests.PickingTaskItemRequest, error) {
	var items []requests.PickingTaskItemRequest
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}

	// Second pass: check for legacy "location" field (items with no allocations).
	var legacyPayload []map[string]interface{}
	if err := json.Unmarshal(raw, &legacyPayload); err != nil {
		// If this fails the first unmarshal was sufficient.
		return items, nil
	}

	for i := range items {
		if len(items[i].Allocations) == 0 && i < len(legacyPayload) {
			if location, ok := legacyPayload[i]["location"].(string); ok && location != "" {
				items[i].Allocations = []database.LocationAllocation{
					{Location: location, Quantity: items[i].ExpectedQuantity},
				}
			}
		}
	}
	return items, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// B3 — Shared reservation helpers
// ─────────────────────────────────────────────────────────────────────────────

// applyReservations increments reserved_qty for every allocation within tx.
// Uses a conditional UPDATE so it fails fast (RowsAffected == 0) when there
// is not enough available stock, without ever leaving inventory in a bad state.
// The caller must propagate a sentinel error to trigger rollback.
func (r *PickingTaskRepository) applyReservations(tx *gorm.DB, items []requests.PickingTaskItemRequest) *responses.InternalResponse {
	for _, item := range items {
		for _, alloc := range item.Allocations {
			result := tx.Exec(`
				UPDATE inventory
				   SET reserved_qty = reserved_qty + ?,
				       updated_at   = NOW()
				 WHERE sku = ? AND location = ?
				   AND (quantity - reserved_qty) >= ?
			`, alloc.Quantity, item.SKU, alloc.Location, alloc.Quantity)

			if result.Error != nil {
				return &responses.InternalResponse{
					Error:   fmt.Errorf("reservar %s @ %s: %w", item.SKU, alloc.Location, result.Error),
					Message: "Error al reservar stock",
					Handled: false,
				}
			}
			if result.RowsAffected == 0 {
				return &responses.InternalResponse{
					Message: fmt.Sprintf(
						"Stock insuficiente para %s en %s. No hay %.2f uds disponibles para reservar.",
						item.SKU, alloc.Location, alloc.Quantity,
					),
					Handled:    true,
					StatusCode: responses.StatusBadRequest,
				}
			}
		}
	}
	return nil
}

// releaseReservations decrements reserved_qty for every allocation within tx.
// Uses GREATEST(0, ...) so releasing more than what is reserved is safe.
func (r *PickingTaskRepository) releaseReservations(tx *gorm.DB, items []requests.PickingTaskItemRequest) *responses.InternalResponse {
	for _, item := range items {
		for _, alloc := range item.Allocations {
			if err := tx.Exec(`
				UPDATE inventory
				   SET reserved_qty = GREATEST(0, reserved_qty - ?),
				       updated_at   = NOW()
				 WHERE sku = ? AND location = ?
			`, alloc.Quantity, item.SKU, alloc.Location).Error; err != nil {
				return &responses.InternalResponse{
					Error:   fmt.Errorf("liberar %s @ %s: %w", item.SKU, alloc.Location, err),
					Message: "Error al liberar reservas",
					Handled: false,
				}
			}
		}
	}
	return nil
}

// validateNoExpiredLots checks the DB for any lot referenced in items that is
// past its expiration date. This queries the DB rather than the request payload
// so stale expiration dates in old drafts are caught correctly.
func (r *PickingTaskRepository) validateNoExpiredLots(items []requests.PickingTaskItemRequest) *responses.InternalResponse {
	today := time.Now().Truncate(24 * time.Hour)

	type key struct{ SKU, Lot string }
	refs := make(map[key]bool)
	for _, item := range items {
		for _, lot := range item.LotNumbers {
			if lot.LotNumber != "" {
				refs[key{item.SKU, lot.LotNumber}] = true
			}
		}
		for _, alloc := range item.Allocations {
			if alloc.LotNumber != nil && *alloc.LotNumber != "" {
				refs[key{item.SKU, *alloc.LotNumber}] = true
			}
		}
	}

	for k := range refs {
		var lot database.Lot
		if err := r.DB.Where("sku = ? AND lot_number = ?", k.SKU, k.Lot).First(&lot).Error; err != nil {
			continue // lot not in DB — validated elsewhere
		}
		if lot.ExpirationDate != nil && lot.ExpirationDate.Before(today) {
			return &responses.InternalResponse{
				Message: fmt.Sprintf(
					"Lote %s (SKU %s) venció el %s. No puede pickearse.",
					k.Lot, k.SKU, lot.ExpirationDate.Format("2006-01-02"),
				),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
	}
	return nil
}

// sanitizePickingUpdatePayload applies the whitelist and key normalisation that
// UpdatePickingTask used inline. Extracted as a helper so it can be reused.
func sanitizePickingUpdatePayload(data map[string]interface{}) map[string]interface{} {
	protected := map[string]bool{
		"id":         true,
		"task_id":    true,
		"created_at": true,
	}
	whitelist := map[string]bool{
		"assigned_to":  true,
		"priority":     true,
		"status":       true,
		"notes":        true,
		"items":        true,
		"order_number": true,
		"updated_at":   true,
		"completed_at": true,
		"customer_id":  true, // S2 R2
	}

	clean := make(map[string]interface{}, len(data)+2)
	for k, v := range data {
		key := strings.ToLower(k)
		key = strings.ReplaceAll(key, "assignedto", "assigned_to")
		key = strings.ReplaceAll(key, "ordernumber", "order_number")
		key = strings.ReplaceAll(key, "outboundnumber", "order_number")
		key = strings.ReplaceAll(key, "completedat", "completed_at")
		key = strings.ReplaceAll(key, "updatedat", "updated_at")

		if protected[key] || !whitelist[key] {
			continue
		}
		clean[key] = v
	}
	return clean
}

// ─────────────────────────────────────────────────────────────────────────────
// Read methods
// ─────────────────────────────────────────────────────────────────────────────

// GetAllPickingTasks returns all picking tasks without tenant filter.
// internal use only — bypass tenant. Prefer GetAllForTenant in HTTP handlers.
func (r *PickingTaskRepository) GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse) {
	var tasks []responses.PickingTaskView

	sqlRar := `
		SELECT
			pt.id,
			pt.task_id,
			pt.order_number,
			pt.created_by,
			usr.first_name || ' ' || usr.last_name AS user_creator_name,
			pt.assigned_to,
			usr_assignee.first_name || ' ' || usr_assignee.last_name AS user_assignee_name,
			pt.status,
			pt.priority,
			pt.notes,
			pt.created_at,
			pt.updated_at,
			pt.completed_at,
			pt.customer_id,
			c.code AS customer_code,
			c.name AS customer_name,
			jsonb_agg(
				jsonb_build_object(
					'sku', item->>'sku',
					'item_name', a.name,
					'status', COALESCE(item->>'status', 'pending'),
					'location', item->>'location',
					'required_qty', item->>'required_qty',
					'picked_qty', item->>'picked_qty',
					'lots', (
						SELECT jsonb_agg(l)
						FROM jsonb_array_elements(item->'lots') AS l
					),
					'serials', (
						SELECT jsonb_agg(s)
						FROM jsonb_array_elements(item->'serials') AS s
					)
				)
			) AS items
		FROM picking_tasks pt
		INNER JOIN users usr ON pt.created_by = usr.id
		LEFT JOIN users usr_assignee ON pt.assigned_to = usr_assignee.id
		LEFT JOIN LATERAL jsonb_array_elements(pt.items) AS item ON TRUE
		LEFT JOIN articles a ON a.sku = item->>'sku'
		LEFT JOIN clients c ON pt.customer_id = c.id
		GROUP BY
			pt.id,
			pt.task_id,
			pt.order_number,
			pt.created_by,
			usr.first_name,
			usr.last_name,
			pt.assigned_to,
			usr_assignee.first_name,
			usr_assignee.last_name,
			pt.status,
			pt.priority,
			pt.notes,
			pt.created_at,
			pt.updated_at,
			pt.completed_at,
			pt.customer_id,
			c.code,
			c.name;
	`

	if err := r.DB.Raw(sqlRar).Scan(&tasks).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener todas las tareas de picking",
			Handled: false,
		}
	}
	return tasks, nil
}

// GetAllForTenant returns picking tasks scoped to a specific tenant (S2.5 M3.1).
func (r *PickingTaskRepository) GetAllForTenant(tenantID string) ([]responses.PickingTaskView, *responses.InternalResponse) {
	var tasks []responses.PickingTaskView

	sqlTenant := `
		SELECT
			pt.id,
			pt.task_id,
			pt.order_number,
			pt.created_by,
			usr.first_name || ' ' || usr.last_name AS user_creator_name,
			pt.assigned_to,
			usr_assignee.first_name || ' ' || usr_assignee.last_name AS user_assignee_name,
			pt.status,
			pt.priority,
			pt.notes,
			pt.created_at,
			pt.updated_at,
			pt.completed_at,
			pt.customer_id,
			c.code AS customer_code,
			c.name AS customer_name,
			jsonb_agg(
				jsonb_build_object(
					'sku', item->>'sku',
					'item_name', a.name,
					'status', COALESCE(item->>'status', 'pending'),
					'location', item->>'location',
					'required_qty', item->>'required_qty',
					'picked_qty', item->>'picked_qty',
					'lots', (
						SELECT jsonb_agg(l)
						FROM jsonb_array_elements(item->'lots') AS l
					),
					'serials', (
						SELECT jsonb_agg(s)
						FROM jsonb_array_elements(item->'serials') AS s
					)
				)
			) AS items
		FROM picking_tasks pt
		INNER JOIN users usr ON pt.created_by = usr.id
		LEFT JOIN users usr_assignee ON pt.assigned_to = usr_assignee.id
		LEFT JOIN LATERAL jsonb_array_elements(pt.items) AS item ON TRUE
		LEFT JOIN articles a ON a.sku = item->>'sku'
		LEFT JOIN clients c ON pt.customer_id = c.id
		WHERE pt.tenant_id = ?
		GROUP BY
			pt.id,
			pt.task_id,
			pt.order_number,
			pt.created_by,
			usr.first_name,
			usr.last_name,
			pt.assigned_to,
			usr_assignee.first_name,
			usr_assignee.last_name,
			pt.status,
			pt.priority,
			pt.notes,
			pt.created_at,
			pt.updated_at,
			pt.completed_at,
			pt.customer_id,
			c.code,
			c.name;
	`

	if err := r.DB.Raw(sqlTenant, tenantID).Scan(&tasks).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener todas las tareas de picking",
			Handled: false,
		}
	}
	return tasks, nil
}

func (r *PickingTaskRepository) GetPickingTaskByID(id string) (*database.PickingTask, *responses.InternalResponse) {
	var task database.PickingTask
	if err := r.DB.Table(database.PickingTask{}.TableName()).Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Tarea de picking no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la tarea de picking",
			Handled: false,
		}
	}
	return &task, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CreatePickingTask
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) CreatePickingTask(userId string, tenantID string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	var items []requests.PickingTaskItemRequest
	if err := json.Unmarshal(task.Items, &items); err != nil {
		*handledResp = responses.InternalResponse{Error: err, Message: "Formato de items inválido", Handled: true}
		return handledResp
	}

	// Validate each item's allocations sum
	for _, it := range items {
		if err := it.ValidateAllocationSum(); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: err.Error(), Handled: true, StatusCode: responses.StatusBadRequest}
			return handledResp
		}
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&database.PickingTask{}).Where("order_number = ? AND tenant_id = ?", task.OutboundNumber, tenantID).Count(&count).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al verificar la unicidad del número de salida", Handled: false}
			return nil
		}
		if count > 0 {
			*handledResp = responses.InternalResponse{
				Error:      fmt.Errorf("outbound number %s is already taken", task.OutboundNumber),
				Message:    "El número de salida ya está en uso",
				Handled:    true,
				StatusCode: responses.StatusConflict,
			}
			return nil
		}

		nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
		taskID := fmt.Sprintf("PICK-%06d", nowMillis%1_000_000)

		articleCache := make(map[string]database.Article)
		for i := range items {
			items[i].Status = tools.StrPtr("open")
			sku := items[i].SKU

			art, ok := articleCache[sku]
			if !ok {
				if err := tx.Where("sku = ?", sku).First(&art).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return fmt.Errorf("article not found for SKU %s", sku)
					}
					return fmt.Errorf("find article %s: %w", sku, err)
				}
				articleCache[sku] = art
			}

			if art.TrackByLot {
				for j := range items[i].LotNumbers {
					items[i].LotNumbers[j].Status = tools.StrPtr("open")
				}
			}
			if art.TrackBySerial {
				for j := range items[i].SerialNumbers {
					items[i].SerialNumbers[j].Status = "open"
				}
			}
		}

		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal items: %w", err)
		}

		priority := task.Priority
		if priority == "" {
			priority = "normal"
		}

		id, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generar id picking task: %w", err)
		}

		pickingTask := database.PickingTask{
			ID:          id,
			TaskID:      taskID,
			OrderNumber: task.OutboundNumber,
			CreatedBy:   userId,
			AssignedTo:  task.AssignedTo,
			Status:      "open",
			Priority:    priority,
			Notes:       task.Notes,
			Items:       json.RawMessage(itemsJSON),
			CustomerID:  task.CustomerID, // S2 R2
			TenantID:    tenantID,        // S2.5 M3.1
		}

		if err := tx.Create(&pickingTask).Error; err != nil {
			return fmt.Errorf("crear tarea de picking: %w", err)
		}
		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error en la transacción"}
	}
	if handledResp.Error != nil || handledResp.Handled {
		return handledResp
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// B3a — StartPickingTask: applies lazy reservations
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) StartPickingTask(ctx context.Context, id, userId string) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.PickingTask
		if err := tx.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handledResp = &responses.InternalResponse{
					Message: "Tarea no encontrada", Handled: true, StatusCode: responses.StatusNotFound,
				}
				return fmt.Errorf("not found")
			}
			return err
		}

		// Only open|assigned → in_progress is a valid start transition.
		if task.Status != "open" && task.Status != "assigned" {
			handledResp = &responses.InternalResponse{
				Message:    fmt.Sprintf("No se puede iniciar una tarea en estado '%s'", task.Status),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
			return fmt.Errorf("invalid transition")
		}

		items, err := parsePickingItemsWithLegacyFallback(task.Items)
		if err != nil {
			return fmt.Errorf("parse items: %w", err)
		}

		// Validate no expired lots before reserving (B4).
		if resp := r.validateNoExpiredLots(items); resp != nil {
			handledResp = resp
			return fmt.Errorf("expired lot")
		}

		// Apply lazy reservations.
		if resp := r.applyReservations(tx, items); resp != nil {
			handledResp = resp
			return fmt.Errorf("reservation failed")
		}

		if err := tx.Exec(
			`UPDATE picking_tasks SET status = 'in_progress', updated_at = NOW() WHERE id = ?`, id,
		).Error; err != nil {
			return fmt.Errorf("update status: %w", err)
		}

		return nil
	})

	if txErr != nil {
		if handledResp != nil {
			return handledResp
		}
		return &responses.InternalResponse{Error: txErr, Message: "Error al iniciar picking"}
	}

	if r.AuditService != nil {
		r.AuditService.Log(ctx, &userId, tools.ActionExecute, "picking_task", id, nil, nil, "", "")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// B3b/B3c — UpdatePickingTask: full rewrite with reserve recalc
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) UpdatePickingTask(ctx context.Context, id string, data map[string]interface{}, userId string) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.PickingTask
		if err := tx.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handledResp = &responses.InternalResponse{
					Message: "Tarea no encontrada", Handled: true, StatusCode: responses.StatusNotFound,
				}
				return fmt.Errorf("not found")
			}
			return err
		}

		clean := sanitizePickingUpdatePayload(data)
		clean["updated_at"] = tools.GetCurrentTime()

		// Validate and apply status transition if status changed.
		nextStatus, _ := clean["status"].(string)
		if nextStatus != "" {
			currentStatus := strings.ToLower(strings.TrimSpace(task.Status))
			nextStatus = strings.ToLower(strings.TrimSpace(nextStatus))
			clean["status"] = nextStatus

			if !isValidPickingTransition(currentStatus, nextStatus) {
				handledResp = &responses.InternalResponse{
					Message:    fmt.Sprintf("Transición inválida: %s → %s", currentStatus, nextStatus),
					Handled:    true,
					StatusCode: responses.StatusConflict,
				}
				return fmt.Errorf("invalid transition")
			}

			// B3c — cancel from in_progress releases reservations.
			if nextStatus == "cancelled" && currentStatus == "in_progress" {
				oldItems, err := parsePickingItemsWithLegacyFallback(task.Items)
				if err != nil {
					return fmt.Errorf("parse old items for cancel: %w", err)
				}
				if resp := r.releaseReservations(tx, oldItems); resp != nil {
					handledResp = resp
					return fmt.Errorf("release failed")
				}
			}

			// Set completed_at based on terminal status.
			switch nextStatus {
			case "completed", "completed_with_differences", "cancelled", "abandoned", "closed":
				clean["completed_at"] = tools.GetCurrentTime()
			default:
				clean["completed_at"] = gorm.Expr("NULL")
			}
		}

		// B3b — if items changed while task is in_progress, recalculate reservations.
		if rawItems, itemsChanged := clean["items"]; itemsChanged && task.Status == "in_progress" {
			// Release old reservations first.
			oldItems, err := parsePickingItemsWithLegacyFallback(task.Items)
			if err != nil {
				return fmt.Errorf("parse old items: %w", err)
			}
			if resp := r.releaseReservations(tx, oldItems); resp != nil {
				handledResp = resp
				return fmt.Errorf("release old failed")
			}

			// Convert clean["items"] ([]interface{} from JSON decode) → typed slice
			// via re-marshal to avoid unsafe type assertions.
			newItemsBytes, err := json.Marshal(rawItems)
			if err != nil {
				return fmt.Errorf("marshal new items: %w", err)
			}
			var newItems []requests.PickingTaskItemRequest
			if err := json.Unmarshal(newItemsBytes, &newItems); err != nil {
				return fmt.Errorf("parse new items: %w", err)
			}

			if resp := r.validateNoExpiredLots(newItems); resp != nil {
				handledResp = resp
				return fmt.Errorf("expired lot in update")
			}

			if resp := r.applyReservations(tx, newItems); resp != nil {
				handledResp = resp
				return fmt.Errorf("apply new reservations failed")
			}

			// Store the serialised items.
			clean["items"] = json.RawMessage(newItemsBytes)
		} else if rawItems, itemsChanged := clean["items"]; itemsChanged {
			// Not in_progress — just marshal correctly.
			newItemsBytes, err := json.Marshal(rawItems)
			if err != nil {
				return fmt.Errorf("marshal items: %w", err)
			}
			clean["items"] = json.RawMessage(newItemsBytes)
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			return fmt.Errorf("update task: %w", err)
		}

		return nil
	})

	if txErr != nil {
		if handledResp != nil {
			return handledResp
		}
		return &responses.InternalResponse{Error: txErr, Message: "Error al actualizar tarea"}
	}

	if r.AuditService != nil {
		r.AuditService.Log(ctx, &userId, tools.ActionUpdate, "picking_task", id, nil, nil, "", "")
	}

	// Emit task_assigned notification if assigned_to changed.
	if r.NotificationsSvc != nil {
		if newAssignee, ok := data["assigned_to"].(string); ok && newAssignee != "" {
			_ = r.NotificationsSvc.Send(ctx, newAssignee, "task_assigned",
				"Nueva tarea de picking asignada", fmt.Sprintf("Se te ha asignado la tarea de picking %s.", id),
				"picking_task", id)
		}
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// B3d — CompletePickingLine: consumes reservations per allocation
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) CompletePickingLine(ctx context.Context, id, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.PickingTask
		if err := tx.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handledResp = &responses.InternalResponse{Message: "Tarea de picking no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
				return fmt.Errorf("not found")
			}
			return err
		}

		existingItems, err := parsePickingItemsWithLegacyFallback(task.Items)
		if err != nil {
			return fmt.Errorf("parse task items: %w", err)
		}

		// Validate lots before touching inventory.
		if resp := r.validateNoExpiredLots([]requests.PickingTaskItemRequest{item}); resp != nil {
			handledResp = resp
			return fmt.Errorf("expired lot in complete line")
		}

		// Find the matching item by SKU.
		foundIdx := -1
		for i := range existingItems {
			if existingItems[i].SKU == item.SKU {
				foundIdx = i
				break
			}
		}
		if foundIdx == -1 {
			handledResp = &responses.InternalResponse{Message: "Item no encontrado en la tarea de picking", Handled: true}
			return fmt.Errorf("item not found")
		}

		// Decrement inventory, reserved_qty, and lot quantities per allocation.
		for _, alloc := range item.Allocations {
			pickedQty := alloc.Quantity
			if alloc.PickedQty != nil {
				pickedQty = *alloc.PickedQty
			}

			// Fetch inventory before decrement to populate before/after qty on movement.
			var inv database.Inventory
			if err := tx.Where("sku = ? AND location = ?", item.SKU, alloc.Location).First(&inv).Error; err != nil {
				return fmt.Errorf("fetch inventory %s @ %s: %w", item.SKU, alloc.Location, err)
			}

			if err := tx.Exec(`
				UPDATE inventory
				   SET quantity     = quantity - ?,
				       reserved_qty = GREATEST(0, reserved_qty - ?),
				       updated_at   = NOW()
				 WHERE sku = ? AND location = ?
			`, pickedQty, alloc.Quantity, item.SKU, alloc.Location).Error; err != nil {
				return fmt.Errorf("decrementar inventario %s @ %s: %w", item.SKU, alloc.Location, err)
			}

			if alloc.LotNumber != nil && *alloc.LotNumber != "" {
				// Decrement inventory_lots (per-location lot quantity).
				tx.Exec(`
					UPDATE inventory_lots
					   SET quantity = GREATEST(0, quantity - ?)
					 WHERE inventory_id = (SELECT id FROM inventory WHERE sku = ? AND location = ? LIMIT 1)
					   AND lot_id      = (SELECT id FROM lots         WHERE sku = ? AND lot_number = ? LIMIT 1)
				`, pickedQty, item.SKU, alloc.Location, item.SKU, *alloc.LotNumber)

				// Decrement lots global quantity.
				tx.Exec(`
					UPDATE lots SET quantity = GREATEST(0, quantity - ?), updated_at = NOW()
					 WHERE sku = ? AND lot_number = ?
				`, pickedQty, item.SKU, *alloc.LotNumber)
			}

			// Resolve lot_id if allocation references a lot.
			var lotID *string
			if alloc.LotNumber != nil && *alloc.LotNumber != "" {
				var lot database.Lot
				if err := tx.Where("sku = ? AND lot_number = ?", item.SKU, *alloc.LotNumber).First(&lot).Error; err == nil {
					lotID = &lot.ID
				}
			}

			// Create OUTBOUND movement (D2 — deuda S1 D4).
			movID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate picking movement id: %w", err)
			}
			beforeQty := inv.Quantity
			afterQty := inv.Quantity - pickedQty
			refType := "picking_task"
			mov := &database.InventoryMovement{
				ID:             movID,
				SKU:            item.SKU,
				Location:       alloc.Location,
				MovementType:   "outbound",
				Quantity:       pickedQty,
				RemainingStock: afterQty,
				CreatedBy:      userId,
				CreatedAt:      tools.GetCurrentTime(),
				ReferenceType:  &refType,
				ReferenceID:    &id,
				LotID:          lotID,
				UnitCost:       inv.UnitPrice,
				BeforeQty:      &beforeQty,
				AfterQty:       &afterQty,
				UserID:         &userId,
			}
			if err := tx.Create(mov).Error; err != nil {
				return fmt.Errorf("create outbound movement %s @ %s: %w", item.SKU, alloc.Location, err)
			}
		}

		// Update item status in the task's JSONB.
		totalPicked := 0.0
		for _, alloc := range item.Allocations {
			if alloc.PickedQty != nil {
				totalPicked += *alloc.PickedQty
			} else {
				totalPicked += alloc.Quantity
			}
		}
		if totalPicked >= existingItems[foundIdx].ExpectedQuantity {
			existingItems[foundIdx].Status = tools.StrPtr("completed")
		} else {
			existingItems[foundIdx].Status = tools.StrPtr("partial")
		}
		// Store updated allocations from the incoming item.
		existingItems[foundIdx].Allocations = item.Allocations

		updatedItems, err := json.Marshal(existingItems)
		if err != nil {
			return fmt.Errorf("marshal updated items: %w", err)
		}

		if err := tx.Exec(
			`UPDATE picking_tasks SET items = ?, updated_at = NOW() WHERE id = ?`,
			json.RawMessage(updatedItems), id,
		).Error; err != nil {
			return fmt.Errorf("update task items: %w", err)
		}

		return nil
	})

	if txErr != nil {
		if handledResp != nil {
			return handledResp
		}
		return &responses.InternalResponse{Error: txErr, Message: "Error al completar línea de picking"}
	}

	if r.AuditService != nil {
		r.AuditService.Log(ctx, &userId, tools.ActionUpdate, "picking_task", id, nil, nil, "", "")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// H5 — CompletePickingTask: full task, allocation-aware
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) CompletePickingTask(ctx context.Context, id, userId string) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.PickingTask
		if err := tx.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handledResp = &responses.InternalResponse{Message: "Tarea no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
				return fmt.Errorf("not found")
			}
			return err
		}

		if task.Status != "in_progress" {
			handledResp = &responses.InternalResponse{
				Message:    fmt.Sprintf("No se puede completar una tarea en estado '%s'", task.Status),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
			return fmt.Errorf("invalid transition")
		}

		items, err := parsePickingItemsWithLegacyFallback(task.Items)
		if err != nil {
			return fmt.Errorf("parse items: %w", err)
		}

		hasDifferences := false
		for _, item := range items {
			for _, alloc := range item.Allocations {
				pickedQty := alloc.Quantity
				if alloc.PickedQty != nil {
					pickedQty = *alloc.PickedQty
				}
				if pickedQty != alloc.Quantity {
					hasDifferences = true
				}

				// Fetch inventory before decrement to populate before/after qty on movement.
				var inv database.Inventory
				if err := tx.Where("sku = ? AND location = ?", item.SKU, alloc.Location).First(&inv).Error; err != nil {
					return fmt.Errorf("fetch inventory %s @ %s: %w", item.SKU, alloc.Location, err)
				}

				// Decrement inventory quantity + release reservation.
				if err := tx.Exec(`
					UPDATE inventory
					   SET quantity     = quantity - ?,
					       reserved_qty = GREATEST(0, reserved_qty - ?),
					       updated_at   = NOW()
					 WHERE sku = ? AND location = ?
				`, pickedQty, alloc.Quantity, item.SKU, alloc.Location).Error; err != nil {
					return fmt.Errorf("decrementar %s @ %s: %w", item.SKU, alloc.Location, err)
				}

				// Decrement lot-level quantities when a specific lot was picked.
				if alloc.LotNumber != nil && *alloc.LotNumber != "" {
					tx.Exec(`
						UPDATE inventory_lots
						   SET quantity = GREATEST(0, quantity - ?)
						 WHERE inventory_id = (SELECT id FROM inventory WHERE sku = ? AND location = ? LIMIT 1)
						   AND lot_id      = (SELECT id FROM lots         WHERE sku = ? AND lot_number = ? LIMIT 1)
					`, pickedQty, item.SKU, alloc.Location, item.SKU, *alloc.LotNumber)

					tx.Exec(`
						UPDATE lots SET quantity = GREATEST(0, quantity - ?), updated_at = NOW()
						 WHERE sku = ? AND lot_number = ?
					`, pickedQty, item.SKU, *alloc.LotNumber)
				}

				// Resolve lot_id if allocation references a lot.
				var lotID *string
				if alloc.LotNumber != nil && *alloc.LotNumber != "" {
					var lot database.Lot
					if err := tx.Where("sku = ? AND lot_number = ?", item.SKU, *alloc.LotNumber).First(&lot).Error; err == nil {
						lotID = &lot.ID
					}
				}

				// Create OUTBOUND movement per allocation (D2 — deuda S1 D4).
				movID, err := tools.GenerateNanoid(tx)
				if err != nil {
					return fmt.Errorf("generate picking movement id: %w", err)
				}
				beforeQty := inv.Quantity
				afterQty := inv.Quantity - pickedQty
				refType := "picking_task"
				mov := &database.InventoryMovement{
					ID:             movID,
					SKU:            item.SKU,
					Location:       alloc.Location,
					MovementType:   "outbound",
					Quantity:       pickedQty,
					RemainingStock: afterQty,
					CreatedBy:      userId,
					CreatedAt:      tools.GetCurrentTime(),
					ReferenceType:  &refType,
					ReferenceID:    &id,
					LotID:          lotID,
					UnitCost:       inv.UnitPrice,
					BeforeQty:      &beforeQty,
					AfterQty:       &afterQty,
					UserID:         &userId,
				}
				if err := tx.Create(mov).Error; err != nil {
					return fmt.Errorf("create outbound movement %s @ %s: %w", item.SKU, alloc.Location, err)
				}
			}
		}

		finalStatus := "completed"
		if hasDifferences {
			finalStatus = "completed_with_differences"
		}

		if err := tx.Exec(
			`UPDATE picking_tasks SET status = ?, completed_at = NOW(), updated_at = NOW() WHERE id = ?`,
			finalStatus, id,
		).Error; err != nil {
			return fmt.Errorf("update status: %w", err)
		}

		return nil
	})

	if txErr != nil {
		if handledResp != nil {
			return handledResp
		}
		return &responses.InternalResponse{Error: txErr, Message: "Error al completar picking"}
	}

	if r.AuditService != nil {
		r.AuditService.Log(ctx, &userId, tools.ActionExecute, "picking_task", id, nil, nil, "", "")
	}

	// Emit task_completed notification to the assigned operator (fire-and-forget).
	if r.NotificationsSvc != nil {
		var task database.PickingTask
		if err := r.DB.Select("assigned_to").First(&task, "id = ?", id).Error; err == nil && task.AssignedTo != nil && *task.AssignedTo != "" {
			_ = r.NotificationsSvc.Send(ctx, *task.AssignedTo, "task_completed",
				"Tarea de picking completada", fmt.Sprintf("La tarea de picking %s ha sido completada.", id),
				"picking_task", id)
		}
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Excel Import / Export
// ─────────────────────────────────────────────────────────────────────────────

func (r *PickingTaskRepository) ImportPickingTaskFromExcel(userID string, tenantID string, fileBytes []byte) *responses.InternalResponse {
	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al abrir el archivo de Excel"}
	}
	defer f.Close()

	const sheet = "Sheet1"
	rows, err := f.GetRows(sheet)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al leer las filas de la hoja de Excel"}
	}
	if len(rows) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("empty sheet"), Message: "El archivo de Excel no contiene datos", Handled: true}
	}

	getOneOf := func(labels ...string) *string {
		for _, row := range rows[:minInt(len(rows), 30)] {
			for j, cell := range row {
				cellNorm := strings.ToLower(strings.TrimSpace(cell))
				for _, lab := range labels {
					if cellNorm == strings.ToLower(strings.TrimSpace(lab)) {
						for k := j + 1; k < len(row); k++ {
							if v := strings.TrimSpace(row[k]); v != "" {
								return ptr(v)
							}
						}
						return ptr("")
					}
				}
			}
		}
		return nil
	}

	outboundNumber := getOneOf("Outbound Number", "Order Number")
	assignedTo := getOneOf("Assigned To")
	priority := getOneOf("Priority")
	notes := getOneOf("Notes")

	var assignedId string
	if assignedTo != nil && strings.TrimSpace(*assignedTo) != "" {
		var user database.User
		if err := r.DB.Where("email = ?", strings.TrimSpace(*assignedTo)).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &responses.InternalResponse{Error: fmt.Errorf("user %s not found", *assignedTo), Message: "Usuario asignado no encontrado", Handled: true}
			}
		} else {
			assignedId = user.ID
		}
	} else {
		return &responses.InternalResponse{Error: fmt.Errorf("assigned to is required"), Message: "El campo 'Assigned To' es obligatorio", Handled: true}
	}

	priorityNorm := "normal"
	if priority != nil && strings.TrimSpace(*priority) != "" {
		switch p := strings.ToLower(strings.TrimSpace(*priority)); p {
		case "low", "baja":
			priorityNorm = "low"
		case "high", "alta":
			priorityNorm = "high"
		}
	}

	headerRowIdx := -1
	colIndex := map[string]int{}

	for i, row := range rows {
		tmp := map[string]int{
			"sku":            -1,
			"qty":            -1,
			"location":       -1,
			"lot_numbers":    -1,
			"serial_numbers": -1,
		}
		found := 0
		for j, cell := range row {
			key := strings.ToLower(strings.TrimSpace(cell))
			switch key {
			case "sku":
				tmp["sku"] = j
				found++
			case "expected quantity", "requested quantity":
				tmp["qty"] = j
				found++
			case "location", "from location":
				tmp["location"] = j
				found++
			case "lot numbers":
				tmp["lot_numbers"] = j
				found++
			case "serial numbers":
				tmp["serial_numbers"] = j
				found++
			}
		}
		if tmp["sku"] >= 0 && (tmp["qty"] >= 0 || tmp["location"] >= 0) {
			headerRowIdx = i
			colIndex = tmp
			break
		}
	}
	if headerRowIdx == -1 {
		return &responses.InternalResponse{
			Error:   fmt.Errorf("headers not found"),
			Message: "Fila de encabezado de items no encontrada (SKU, Cantidad Esperada/Solicitada, Ubicación, Números de Lote, Números de Serie)",
			Handled: true,
		}
	}

	var items []requests.PickingTaskItemRequest

	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]

		sku := get(row, colIndex["sku"])
		if strings.TrimSpace(sku) == "" {
			continue
		}

		qtyStr := get(row, colIndex["qty"])
		location := get(row, colIndex["location"])
		lotsStr := get(row, colIndex["lot_numbers"])
		serialsStr := get(row, colIndex["serial_numbers"])

		qty := 0.0
		if n, err := strconv.ParseFloat(strings.TrimSpace(qtyStr), 64); err == nil {
			qty = n
		}

		// Build a single allocation from the Excel location column (legacy-style).
		allocations := []database.LocationAllocation{}
		if loc := strings.TrimSpace(location); loc != "" && qty > 0 {
			allocations = append(allocations, database.LocationAllocation{
				Location: loc,
				Quantity: qty,
			})
		}

		// Parse lot numbers from comma-separated string.
		var lotEntries []database.LotEntry
		for _, ln := range splitCSV(lotsStr) {
			if ln != "" {
				lotEntries = append(lotEntries, database.LotEntry{LotNumber: ln, SKU: sku, Quantity: qty})
			}
		}

		// Serial numbers — stored in SerialNumbers field.
		var serials []database.Serial
		for _, sn := range splitCSV(serialsStr) {
			if sn != "" {
				serials = append(serials, database.Serial{SerialNumber: sn, SKU: sku})
			}
		}

		item := requests.PickingTaskItemRequest{
			SKU:              strings.TrimSpace(sku),
			ExpectedQuantity: qty,
			Allocations:      allocations,
		}
		if len(lotEntries) > 0 {
			item.LotNumbers = lotEntries
		}
		if len(serials) > 0 {
			item.SerialNumbers = serials
		}

		if qty > 0 {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("no items"), Message: "No se encontraron items para importar", Handled: true}
	}

	itemsJSON, _ := json.Marshal(items)
	req := &requests.CreatePickingTaskRequest{
		OutboundNumber: safeDeref(outboundNumber),
		AssignedTo:     &assignedId,
		Priority:       priorityNorm,
		Notes:          notes,
		Items:          json.RawMessage(itemsJSON),
	}

	if resp := r.CreatePickingTask(userID, tenantID, req); resp != nil && resp.Error != nil {
		return resp
	}

	return &responses.InternalResponse{
		Message: "Tarea de picking importada y creada con éxito",
		Handled: true,
	}
}

func (r *PickingTaskRepository) ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse) {
	tasks, errResp := r.GetAllPickingTasks()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "Task ID", "Order Number", "Created By", "Assigned To",
		"Status", "Priority", "Notes", "Items", "Created At", "Updated At", "Completed At",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, task := range tasks {
		rowNum := i + 2
		row := []interface{}{
			task.ID, task.TaskID, task.OrderNumber, task.CreatedBy, task.AssignedTo,
			task.Status, task.Priority, task.Notes, string(task.Items),
			task.CreatedAt.Format(time.RFC3339), nil, nil,
		}
		if !task.UpdatedAt.IsZero() {
			row[10] = task.UpdatedAt.Format(time.RFC3339)
		}
		if !task.CompletedAt.IsZero() {
			row[11] = task.CompletedAt.Format(time.RFC3339)
		}
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al generar el archivo de Excel"}
	}
	return buf.Bytes(), nil
}

func (r *PickingTaskRepository) GenerateImportTemplate(language string) ([]byte, error) {
	isEs := language != "en"
	l2 := getLang(language)
	_, _ = l2["yes"], l2["no"]
	title := "Importar Tareas de Picking"
	subtitle := "Plantilla de importación — eSTOCK"
	instrTitle := "📋 Instrucciones"
	instrContent := "1. Complete desde la fila 9  •  2. SKU, Cantidad Solicitada, Ubicación y Asignado A son obligatorios (*)  •  3. Lotes y seriales: separe con comas"
	if !isEs {
		title = "Import Picking Tasks"
		subtitle = "Picking task import template — eSTOCK"
		instrTitle = "📋 Instructions"
		instrContent = "1. Fill in data from row 9  •  2. SKU, Requested Quantity, Location and Assigned To are required (*)  •  3. Lots and serials: separate with commas"
	}
	prios := []string{"normal", "low", "high"}

	cfg := ModuleTemplateConfig{
		DataSheetName: func() string {
			if isEs {
				return "Picking"
			}
			return "PickingTasks"
		}(),
		OptSheetName: func() string {
			if isEs {
				return "Opciones"
			}
			return "Options"
		}(),
		Title: title, Subtitle: subtitle, InstrTitle: instrTitle, InstrContent: instrContent,
		Columns: func() []ColumnDef {
			if isEs {
				return []ColumnDef{
					{Header: "SKU *", Required: true, Width: 14},
					{Header: "Cantidad Solicitada *", Required: true, Width: 20},
					{Header: "Ubicación *", Required: true, Width: 18},
					{Header: "Números de Lote", Required: false, Width: 22},
					{Header: "Números de Serie", Required: false, Width: 22},
					{Header: "Número de Orden", Required: false, Width: 18},
					{Header: "Asignado A *", Required: true, Width: 24},
					{Header: "Prioridad", Required: false, Width: 14},
					{Header: "Notas", Required: false, Width: 28},
				}
			}
			return []ColumnDef{
				{Header: "SKU *", Required: true, Width: 14},
				{Header: "Requested Quantity *", Required: true, Width: 20},
				{Header: "Location *", Required: true, Width: 18},
				{Header: "Lot Numbers", Required: false, Width: 22},
				{Header: "Serial Numbers", Required: false, Width: 22},
				{Header: "Order Number", Required: false, Width: 18},
				{Header: "Assigned To *", Required: true, Width: 24},
				{Header: "Priority", Required: false, Width: 14},
				{Header: "Notes", Required: false, Width: 28},
			}
		}(),
		ExampleRow: []string{"SKU-0001", "25", "LOC-001", "", "", "ORD-001", "operator@company.com", "normal", ""},
		ApplyValidations: func(f *excelize.File, dataSheet, optSheet string, start, end int) error {
			f.NewSheet(optSheet)
			for i, v := range prios {
				cell, _ := excelize.CoordinatesToCellName(1, i+1)
				f.SetCellValue(optSheet, cell, v)
			}
			f.SetSheetVisible(optSheet, false)
			prioRef := "'" + optSheet + "'!$A$1:$A$3"
			errPrio := func() string {
				if isEs {
					return "Prioridad inválida"
				}
				return "Invalid priority"
			}()
			return addDropListValidation(f, dataSheet, "H9:H2000", prioRef, errPrio, errPrio)
		},
	}
	return BuildModuleImportTemplate(cfg)
}

// LinkCustomer links or unlinks a customer on a picking task (S2 R2 E1.7 — Track A).
func (r *PickingTaskRepository) LinkCustomer(taskID string, customerID *string) *responses.InternalResponse {
	update := map[string]interface{}{"customer_id": customerID, "updated_at": tools.GetCurrentTime()}
	if err := r.DB.Model(&database.PickingTask{}).Where("id = ?", taskID).Updates(update).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al vincular cliente"}
	}
	return nil
}
