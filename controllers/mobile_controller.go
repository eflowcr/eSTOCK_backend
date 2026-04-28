// Package-level note: MobileController is the thin facade that mobile clients
// hit at /api/mobile/*. It does NOT duplicate business logic — every method
// reuses an existing service. Mobile-specific concerns it owns:
//   - JWT-based health endpoint
//   - Trimmed list DTOs (drop heavy jsonb columns from list view payloads)
//   - JSON-body line-complete endpoints (mirror of existing URL-param endpoints)
//   - Reading filters from query string instead of route params for cleaner mobile UX
package controllers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// W0.7 N1-2 fix: backend-enforced over-pick tolerance (5%). Mobile used to
// hardcode this client-side as theater because backend never saw the real
// expected qty. With the new MobilePickingTaskDetailDto / MobileCompleteLineRequest
// contract the backend recovers ExpectedQty per line and rejects pick-overs.
const mobilePickingOverTolerance = 1.05

// W7 N1-B fix: mirror the picking tolerance for receiving. Same constant value
// keeps the operator UX consistent across modules — when this needs to diverge
// per-tenant, split into mobilePickingOverTolerance / mobileReceivingOverTolerance
// and wire from configuration.
const mobileReceivingOverTolerance = 1.05

type MobileController struct {
	Picking            *services.PickingTaskService
	Receiving          *services.ReceivingTasksService
	StockTransfers     *services.StockTransfersService
	Inventory          *services.InventoryService
	InventoryMovements *services.InventoryMovementsService
	StockAlerts        *services.StockAlertsService
	Config             configuration.Config
}

func NewMobileController(
	picking *services.PickingTaskService,
	receiving *services.ReceivingTasksService,
	transfers *services.StockTransfersService,
	inventory *services.InventoryService,
	movements *services.InventoryMovementsService,
	alerts *services.StockAlertsService,
	config configuration.Config,
) *MobileController {
	return &MobileController{
		Picking:            picking,
		Receiving:          receiving,
		StockTransfers:     transfers,
		Inventory:          inventory,
		InventoryMovements: movements,
		StockAlerts:        alerts,
		Config:             config,
	}
}

// ─── Health ──────────────────────────────────────────────────────────────────

// Health returns user/tenant/role info from the JWT plus server time + version.
// Mobile clients call this on cold-start to verify token validity before showing the home screen.
func (c *MobileController) Health(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileHealth", "Token inválido", "mobile_health")
		return
	}
	role, _ := tools.GetRole(c.Config.JWTSecret, token)
	userName, _ := tools.GetUserName(c.Config.JWTSecret, token)
	email, _ := tools.GetEmail(c.Config.JWTSecret, token)

	version := c.Config.Version
	if version == "" {
		version = "dev"
	}

	resp := responses.MobileHealthResponse{
		Tenant:     "", // single-tenant codebase; reserved for future
		User:       userID,
		UserName:   userName,
		Email:      email,
		Role:       role,
		ServerTime: time.Now().UTC(),
		Version:    version,
	}
	tools.ResponseOK(ctx, "MobileHealth", "OK", "mobile_health", resp, false, "")
}

// ─── Picking ─────────────────────────────────────────────────────────────────

// ListPickingTasks supports `assigned_to_me=true` and `status=pending,in_progress` filters.
// Filtering is applied in-memory after fetching from the existing service (the underlying
// repo doesn't yet expose filtered list APIs; this is fine for mobile MVP volume).
func (c *MobileController) ListPickingTasks(ctx *gin.Context) {
	tasks, resp := c.Picking.GetAllPickingTasks()
	if resp != nil {
		writeErrorResponse(ctx, "MobileListPickingTasks", "mobile_list_picking_tasks", resp)
		return
	}

	assignedToMe := strings.EqualFold(ctx.Query("assigned_to_me"), "true")
	statusFilter := parseCSVFilter(ctx.Query("status"))

	// W7 N2-1: operators are forced to assigned_to_me=true regardless of the
	// query param to prevent a cross-operator data leak (operator A could
	// previously list every task in the warehouse including those assigned to
	// operator B). Admin/warehouse roles can still opt out by omitting the flag.
	token := ctx.Request.Header.Get("Authorization")
	role, _ := tools.GetRole(c.Config.JWTSecret, token)
	if !assignedToMe && tools.IsOperatorRole(role) {
		assignedToMe = true
	}

	var userID string
	if assignedToMe {
		uid, err := tools.GetUserId(c.Config.JWTSecret, token)
		if err != nil {
			tools.ResponseUnauthorized(ctx, "MobileListPickingTasks", "Token inválido", "mobile_list_picking_tasks")
			return
		}
		userID = uid
	}

	out := make([]responses.MobilePickingTaskSummary, 0, len(tasks))
	for _, t := range tasks {
		if assignedToMe && (t.AssignedTo == nil || *t.AssignedTo != userID) {
			continue
		}
		if len(statusFilter) > 0 && !statusFilter[strings.ToLower(t.Status)] {
			continue
		}
		out = append(out, responses.MobilePickingTaskSummary{
			ID:           t.ID,
			TaskID:       t.TaskID,
			OrderNumber:  t.OrderNumber,
			Status:       t.Status,
			Priority:     t.Priority,
			AssignedTo:   t.AssignedTo,
			AssigneeName: t.UserAssigneeName,
			CreatedAt:    t.CreatedAt,
			CompletedAt:  t.CompletedAt,
		})
	}
	tools.ResponseOK(ctx, "MobileListPickingTasks", "Tareas de picking obtenidas", "mobile_list_picking_tasks", out, false, "")
}

// GetPickingTask returns the trimmed MobilePickingTaskDetailDto with flat lines.
//
// W0.7 N1-1 fix: previously the controller returned database.PickingTask raw,
// whose items jsonb shape is []PickingTaskItem with nested Allocations. Mobile
// modeled `location` as a flat string per-line that was permanently empty. The
// detail DTO now flattens each item to a single MobilePickingLineDto with the
// location resolved from allocations[0].location, plus a deterministic LineID
// the client echoes back into CompletePickingLine for tolerance validation.
func (c *MobileController) GetPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileGetPickingTask", "mobile_get_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	task, resp := c.Picking.GetPickingTaskByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "MobileGetPickingTask", "mobile_get_picking_task", resp)
		return
	}
	if task == nil {
		tools.ResponseNotFound(ctx, "MobileGetPickingTask", "Tarea no encontrada", "mobile_get_picking_task")
		return
	}
	dto, err := buildMobilePickingTaskDetail(task)
	if err != nil {
		tools.ResponseInternal(ctx, "MobileGetPickingTask", "Error parseando items: "+err.Error(), "mobile_get_picking_task")
		return
	}
	tools.ResponseOK(ctx, "MobileGetPickingTask", "Tarea obtenida", "mobile_get_picking_task", dto, false, "")
}

// buildMobilePickingTaskDetail flattens a database.PickingTask into the
// mobile-facing detail DTO. Items are decoded from the jsonb column and each
// item becomes one MobilePickingLineDto with the first allocation's location
// resolved + sum of picked_qty across allocations.
//
// TODO: multi-allocation (split picking) currently surfaces only the first
// allocation's location to the mobile UI. When the W2/W3 wave needs split-pick
// UX, change this to emit one line per allocation (with the same SKU but
// different location/line_id) so the operator confirms each pick separately.
func buildMobilePickingTaskDetail(task *database.PickingTask) (*responses.MobilePickingTaskDetailDto, error) {
	dto := &responses.MobilePickingTaskDetailDto{
		ID:          task.ID,
		TaskID:      task.TaskID,
		OrderNumber: task.OrderNumber,
		Status:      task.Status,
		Priority:    task.Priority,
		AssignedTo:  task.AssignedTo,
		CompletedAt: task.CompletedAt,
		Lines:       []responses.MobilePickingLineDto{},
	}
	if !task.CreatedAt.IsZero() {
		ca := task.CreatedAt
		dto.CreatedAt = &ca
	}

	if len(task.Items) == 0 {
		return dto, nil
	}
	var items []requests.PickingTaskItemRequest
	if err := json.Unmarshal(task.Items, &items); err != nil {
		return nil, err
	}
	dto.Lines = make([]responses.MobilePickingLineDto, 0, len(items))
	for _, it := range items {
		dto.Lines = append(dto.Lines, mapItemToMobileLine(it))
	}
	return dto, nil
}

// mapItemToMobileLine projects a single PickingTaskItemRequest to the mobile
// flat line shape. PickedQty is summed across allocations (covers both the
// single-allocation case and the future multi-alloc case correctly). When
// allocations is empty (legacy data) location stays "" — the operator will be
// prompted to scan the location at confirm time, but the empty-state used to
// be silent in the UI per W1 hostile review.
func mapItemToMobileLine(it requests.PickingTaskItemRequest) responses.MobilePickingLineDto {
	location := ""
	if len(it.Allocations) > 0 {
		location = it.Allocations[0].Location
	}
	pickedQty := sumAllocationPicked(it)
	lot := ""
	if len(it.LotNumbers) > 0 {
		lot = it.LotNumbers[0].LotNumber
	}
	serial := ""
	if len(it.SerialNumbers) > 0 {
		serial = it.SerialNumbers[0].SerialNumber
	}
	status := "pending"
	if pickedQty >= it.ExpectedQuantity && it.ExpectedQuantity > 0 {
		status = "done"
	} else if pickedQty > 0 {
		status = "partial"
	}
	return responses.MobilePickingLineDto{
		LineID:      computePickingLineID(it.SKU, lot, serial, location),
		SKU:         it.SKU,
		ExpectedQty: it.ExpectedQuantity,
		PickedQty:   pickedQty,
		Status:      status,
		Location:    location,
		Lot:         lot,
		Serial:      serial,
	}
}

// sumAllocationPicked returns the total picked qty across all allocations of an
// item. If no allocation has a PickedQty pointer set we fall back to the
// item-level PickedQty (legacy non-split flow); when neither is populated it
// returns 0.
func sumAllocationPicked(it requests.PickingTaskItemRequest) float64 {
	total := 0.0
	any := false
	for _, a := range it.Allocations {
		if a.PickedQty != nil {
			total += *a.PickedQty
			any = true
		}
	}
	if any {
		return total
	}
	if it.PickedQty != nil {
		return *it.PickedQty
	}
	return 0
}

// computePickingLineID returns a stable 12-char hex hash of the line's
// identifying tuple. Determinism guarantees that GET → POST round-trip works
// without needing to persist a line_id column on the picking_task_items jsonb.
func computePickingLineID(sku, lot, serial, location string) string {
	h := sha1.Sum([]byte(strings.ToLower(sku) + "|" + lot + "|" + serial + "|" + location))
	return hex.EncodeToString(h[:6])
}

// findPickingItemForRequest returns the matching PickingTaskItemRequest from
// the task's items jsonb for a CompletePickingLine request. Match order:
//  1. body.LineID against computePickingLineID(sku, lot, serial, allocations[0].location)
//  2. fallback: (sku, lot, serial) tuple match (case-insensitive on sku)
//
// Returns ok=false when nothing matches. Note that two items with the same
// (sku, lot, serial) tuple is permitted by the schema (different locations);
// the LineID disambiguates that case. Without LineID we take the first match.
func findPickingItemForRequest(task *database.PickingTask, body responses.MobileCompleteLineRequest) (requests.PickingTaskItemRequest, bool) {
	var items []requests.PickingTaskItemRequest
	if len(task.Items) == 0 {
		return requests.PickingTaskItemRequest{}, false
	}
	if err := json.Unmarshal(task.Items, &items); err != nil {
		return requests.PickingTaskItemRequest{}, false
	}
	for _, it := range items {
		line := mapItemToMobileLine(it)
		if body.LineID != "" && line.LineID == body.LineID {
			return it, true
		}
	}
	if body.LineID != "" {
		// LineID supplied but no match — explicit miss, do not fall back.
		return requests.PickingTaskItemRequest{}, false
	}
	for _, it := range items {
		if !strings.EqualFold(it.SKU, body.SKU) {
			continue
		}
		itemLot := ""
		if len(it.LotNumbers) > 0 {
			itemLot = it.LotNumbers[0].LotNumber
		}
		itemSerial := ""
		if len(it.SerialNumbers) > 0 {
			itemSerial = it.SerialNumbers[0].SerialNumber
		}
		if itemLot == body.Lot && itemSerial == body.Serial {
			return it, true
		}
	}
	return requests.PickingTaskItemRequest{}, false
}

func (c *MobileController) StartPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileStartPickingTask", "mobile_start_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	// W0.6: dev sprint-s2 added (ctx, userId) to UpdatePickingTask signature.
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileStartPickingTask", "Token inválido", "invalid_token")
		return
	}
	resp := c.Picking.UpdatePickingTask(ctx.Request.Context(), id, map[string]interface{}{"status": "in_progress"}, userID)
	if resp != nil {
		writeErrorResponse(ctx, "MobileStartPickingTask", "mobile_start_picking_task", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileStartPickingTask", "Tarea iniciada", "mobile_start_picking_task", nil, false, "")
}

// CompletePickingLine accepts JSON body {line_id, sku, picked_qty, location_scanned, lot, serial}
// and forwards to the existing CompletePickingLine service.
//
// W0.7 N1-2 fix: backend recovers the real ExpectedQuantity from the persisted
// task (matching by line_id, falling back to (sku, lot, serial) tuple) and
// validates picked_qty <= expected_qty * 1.05 server-side. Pre-W0.7 the
// controller synthesized ExpectedQuantity = picked_qty so any client could
// over-pick at will and the 5% tolerance was client-only theater.
func (c *MobileController) CompletePickingLine(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileCompletePickingLine", "mobile_complete_picking_line", "ID de tarea inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileCompletePickingLine", "Token inválido", "invalid_token")
		return
	}

	var body responses.MobileCompleteLineRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "MobileCompletePickingLine", "Cuerpo inválido", "mobile_complete_picking_line")
		return
	}
	if strings.TrimSpace(body.LocationScanned) == "" {
		tools.ResponseBadRequest(ctx, "MobileCompletePickingLine", "location_scanned es requerido", "mobile_complete_picking_line")
		return
	}
	if strings.TrimSpace(body.SKU) == "" {
		tools.ResponseBadRequest(ctx, "MobileCompletePickingLine", "sku es requerido", "mobile_complete_picking_line")
		return
	}
	if body.PickedQty <= 0 {
		tools.ResponseBadRequest(ctx, "MobileCompletePickingLine", "picked_qty debe ser mayor a 0", "mobile_complete_picking_line")
		return
	}

	// W0.7: load the task to recover the real ExpectedQuantity for the line.
	task, resp := c.Picking.GetPickingTaskByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompletePickingLine", "mobile_complete_picking_line", resp)
		return
	}
	if task == nil {
		tools.ResponseNotFound(ctx, "MobileCompletePickingLine", "Tarea no encontrada", "mobile_complete_picking_line")
		return
	}
	persisted, found := findPickingItemForRequest(task, body)
	if !found {
		tools.ResponseBadRequest(ctx, "MobileCompletePickingLine", "Línea no encontrada en la tarea", "mobile_complete_picking_line_unknown")
		return
	}
	expectedQty := persisted.ExpectedQuantity
	if expectedQty > 0 && body.PickedQty > expectedQty*mobilePickingOverTolerance {
		tools.ResponseBadRequest(
			ctx,
			"MobileCompletePickingLine",
			"picked_qty excede la tolerancia permitida (5% sobre la cantidad esperada)",
			"mobile_complete_picking_line_tolerance",
		)
		return
	}

	// W0.6: dev sprint-s2 (S1 A1 cross-location) replaced PickingTaskItemRequest.Location
	// with []LocationAllocation. For mobile single-location pick we synthesize a single
	// allocation matching LocationScanned + PickedQty.
	// W0.7: ExpectedQuantity now uses the real persisted value (not the synthesized
	// = picked_qty hack) so downstream validation sees the right anchor.
	pickedQty := body.PickedQty
	item := requests.PickingTaskItemRequest{
		SKU:              body.SKU,
		ExpectedQuantity: expectedQty,
		Allocations: []database.LocationAllocation{{
			Location:  body.LocationScanned,
			Quantity:  pickedQty,
			PickedQty: &pickedQty,
		}},
		PickedQty: &pickedQty,
	}
	if body.Lot != "" {
		item.LotNumbers = []database.LotEntry{{
			LotNumber: body.Lot,
			SKU:       body.SKU,
			Quantity:  pickedQty,
		}}
	}
	if body.Serial != "" {
		item.SerialNumbers = []database.Serial{{SerialNumber: body.Serial, SKU: body.SKU}}
	}

	// W0.6: CompletePickingLine signature is (ctx, id, userId, item). location_scanned
	// is now embedded in item.Allocations.
	resp = c.Picking.CompletePickingLine(ctx.Request.Context(), id, userID, item)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompletePickingLine", "mobile_complete_picking_line", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileCompletePickingLine", "Línea completada", "mobile_complete_picking_line", nil, false, "")
}

// CompletePickingTask requires no body. The W0.6 PickingTaskService signature
// dropped location_scanned (lines complete with their own per-allocation
// location validation), so the mobile endpoint accepts an empty body.
//
// W0.7 Fix D: previously the handler 400'd when location_scanned was missing,
// even though it was discarded server-side. Mobile clients can now POST
// {} or no body at all. The MobileCompleteTaskRequest field is preserved
// (omitempty) for backward compatibility with older clients still sending it.
func (c *MobileController) CompletePickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileCompletePickingTask", "mobile_complete_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileCompletePickingTask", "Token inválido", "invalid_token")
		return
	}
	// Body is optional; ignore parse errors when empty.
	var body responses.MobileCompleteTaskRequest
	_ = ctx.ShouldBindJSON(&body)
	_ = body.LocationScanned // accepted for backward-compat but not consumed

	resp := c.Picking.CompletePickingTask(ctx.Request.Context(), id, userID)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompletePickingTask", "mobile_complete_picking_task", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileCompletePickingTask", "Tarea completada", "mobile_complete_picking_task", nil, false, "")
}

// ─── Receiving ───────────────────────────────────────────────────────────────

func (c *MobileController) ListReceivingTasks(ctx *gin.Context) {
	tasks, resp := c.Receiving.GetAllReceivingTasks()
	if resp != nil {
		writeErrorResponse(ctx, "MobileListReceivingTasks", "mobile_list_receiving_tasks", resp)
		return
	}

	assignedToMe := strings.EqualFold(ctx.Query("assigned_to_me"), "true")
	statusFilter := parseCSVFilter(ctx.Query("status"))

	// W7 N2-1: operators forced to assigned_to_me=true. See ListPickingTasks.
	token := ctx.Request.Header.Get("Authorization")
	role, _ := tools.GetRole(c.Config.JWTSecret, token)
	if !assignedToMe && tools.IsOperatorRole(role) {
		assignedToMe = true
	}

	var userID string
	if assignedToMe {
		uid, err := tools.GetUserId(c.Config.JWTSecret, token)
		if err != nil {
			tools.ResponseUnauthorized(ctx, "MobileListReceivingTasks", "Token inválido", "mobile_list_receiving_tasks")
			return
		}
		userID = uid
	}

	out := make([]responses.MobileReceivingTaskSummary, 0, len(tasks))
	for _, t := range tasks {
		if assignedToMe && (t.AssignedTo == nil || *t.AssignedTo != userID) {
			continue
		}
		if len(statusFilter) > 0 && !statusFilter[strings.ToLower(t.Status)] {
			continue
		}
		out = append(out, responses.MobileReceivingTaskSummary{
			ID:            t.ID,
			TaskID:        t.TaskID,
			InboundNumber: t.InboundNumber,
			Status:        t.Status,
			Priority:      t.Priority,
			AssignedTo:    t.AssignedTo,
			AssigneeName:  t.UserAssigneeName,
			CreatedAt:     t.CreatedAt,
			CompletedAt:   t.CompletedAt,
		})
	}
	tools.ResponseOK(ctx, "MobileListReceivingTasks", "Tareas de recepción obtenidas", "mobile_list_receiving_tasks", out, false, "")
}

// GetReceivingTask returns the trimmed MobileReceivingTaskDetailDto with flat lines.
//
// W7 N1-B fix: mirrors W0.7's picking contract. Previously the endpoint
// returned database.ReceivingTask raw with the `items` jsonb shape exposed —
// the mobile client could not echo a stable line_id and the backend had no
// way to recover ExpectedQuantity for tolerance validation.
func (c *MobileController) GetReceivingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileGetReceivingTask", "mobile_get_receiving_task", "ID de tarea inválido")
	if !ok {
		return
	}
	task, resp := c.Receiving.GetReceivingTaskByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "MobileGetReceivingTask", "mobile_get_receiving_task", resp)
		return
	}
	if task == nil {
		tools.ResponseNotFound(ctx, "MobileGetReceivingTask", "Tarea no encontrada", "mobile_get_receiving_task")
		return
	}
	dto, err := buildMobileReceivingTaskDetail(task)
	if err != nil {
		tools.ResponseInternal(ctx, "MobileGetReceivingTask", "Error parseando items: "+err.Error(), "mobile_get_receiving_task")
		return
	}
	tools.ResponseOK(ctx, "MobileGetReceivingTask", "Tarea obtenida", "mobile_get_receiving_task", dto, false, "")
}

// buildMobileReceivingTaskDetail flattens a database.ReceivingTask into the
// mobile-facing detail DTO. Mirrors buildMobilePickingTaskDetail (W0.7) — see
// that function for the rationale on flattening + LineID determinism.
//
// Receiving items have a single Location field (no allocations array) so the
// surface is simpler than picking — no multi-allocation TODO needed here.
func buildMobileReceivingTaskDetail(task *database.ReceivingTask) (*responses.MobileReceivingTaskDetailDto, error) {
	dto := &responses.MobileReceivingTaskDetailDto{
		ID:          task.ID,
		TaskID:      task.TaskID,
		OrderNumber: task.InboundNumber,
		Status:      task.Status,
		Priority:    task.Priority,
		AssignedTo:  task.AssignedTo,
		CompletedAt: task.CompletedAt,
		Lines:       []responses.MobileReceivingLineDto{},
	}
	if !task.CreatedAt.IsZero() {
		ca := task.CreatedAt
		dto.CreatedAt = &ca
	}
	if len(task.Items) == 0 {
		return dto, nil
	}
	var items []database.ReceivingTaskItem
	if err := json.Unmarshal(task.Items, &items); err != nil {
		return nil, err
	}
	dto.Lines = make([]responses.MobileReceivingLineDto, 0, len(items))
	for _, it := range items {
		dto.Lines = append(dto.Lines, mapReceivingItemToMobileLine(it))
	}
	return dto, nil
}

// mapReceivingItemToMobileLine projects one ReceivingTaskItem to the mobile
// flat shape. ReceivedQty is derived: prefer accepted_qty (S2 R1 explicit
// split), fall back to received_qty pointer for legacy data.
func mapReceivingItemToMobileLine(it database.ReceivingTaskItem) responses.MobileReceivingLineDto {
	receivedQty := it.AcceptedQty
	if receivedQty == 0 && it.ReceivedQuantity != nil {
		receivedQty = *it.ReceivedQuantity
	}
	lot := ""
	if len(it.LotNumbers) > 0 {
		lot = it.LotNumbers[0].LotNumber
	}
	serial := ""
	if len(it.SerialNumbers) > 0 {
		serial = it.SerialNumbers[0].SerialNumber
	}
	status := "pending"
	if it.ExpectedQuantity > 0 && receivedQty >= it.ExpectedQuantity {
		status = "done"
	} else if receivedQty > 0 {
		status = "partial"
	}
	return responses.MobileReceivingLineDto{
		LineID:      computePickingLineID(it.SKU, lot, serial, it.Location),
		SKU:         it.SKU,
		ExpectedQty: it.ExpectedQuantity,
		ReceivedQty: receivedQty,
		Status:      status,
		Location:    it.Location,
		Lot:         lot,
		Serial:      serial,
	}
}

// findReceivingItemForRequest mirrors findPickingItemForRequest (W0.7). Returns
// the matching ReceivingTaskItem so the caller can recover the persisted
// ExpectedQuantity for tolerance validation. Match order:
//  1. body.LineID against computePickingLineID(sku, lot, serial, location)
//  2. fallback: (sku, lot, serial) tuple match (case-insensitive on sku)
//
// LineID supplied + no match → explicit miss (caller 400s; never silently fall
// back, that would mask client/server contract drift).
func findReceivingItemForRequest(task *database.ReceivingTask, body responses.MobileCompleteLineRequest) (database.ReceivingTaskItem, bool) {
	if len(task.Items) == 0 {
		return database.ReceivingTaskItem{}, false
	}
	var items []database.ReceivingTaskItem
	if err := json.Unmarshal(task.Items, &items); err != nil {
		return database.ReceivingTaskItem{}, false
	}
	for _, it := range items {
		line := mapReceivingItemToMobileLine(it)
		if body.LineID != "" && line.LineID == body.LineID {
			return it, true
		}
	}
	if body.LineID != "" {
		return database.ReceivingTaskItem{}, false
	}
	for _, it := range items {
		if !strings.EqualFold(it.SKU, body.SKU) {
			continue
		}
		itemLot := ""
		if len(it.LotNumbers) > 0 {
			itemLot = it.LotNumbers[0].LotNumber
		}
		itemSerial := ""
		if len(it.SerialNumbers) > 0 {
			itemSerial = it.SerialNumbers[0].SerialNumber
		}
		if itemLot == body.Lot && itemSerial == body.Serial {
			return it, true
		}
	}
	return database.ReceivingTaskItem{}, false
}

// CompleteReceivingLine validates the request against the persisted task and
// delegates to the receiving service.
//
// W7 N1-B fix: mirrors W0.7 picking. Pre-W7 the controller synthesized
// ExpectedQuantity = int(body.PickedQty), so any client could over-receive at
// will and tolerance validation was impossible server-side. Now the backend
// recovers the real ExpectedQuantity per line via findReceivingItemForRequest
// and rejects PickedQty > expected_qty * mobileReceivingOverTolerance.
func (c *MobileController) CompleteReceivingLine(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileCompleteReceivingLine", "mobile_complete_receiving_line", "ID de tarea inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileCompleteReceivingLine", "Token inválido", "invalid_token")
		return
	}
	var body responses.MobileCompleteLineRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingLine", "Cuerpo inválido", "mobile_complete_receiving_line")
		return
	}
	if strings.TrimSpace(body.LocationScanned) == "" {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingLine", "location_scanned es requerido", "mobile_complete_receiving_line")
		return
	}
	if strings.TrimSpace(body.SKU) == "" {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingLine", "sku es requerido", "mobile_complete_receiving_line")
		return
	}
	if body.PickedQty <= 0 {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingLine", "received_qty debe ser mayor a 0", "mobile_complete_receiving_line")
		return
	}

	// W7 N1-B: load the task to recover the real ExpectedQuantity for the line.
	task, resp := c.Receiving.GetReceivingTaskByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompleteReceivingLine", "mobile_complete_receiving_line", resp)
		return
	}
	if task == nil {
		tools.ResponseNotFound(ctx, "MobileCompleteReceivingLine", "Tarea no encontrada", "mobile_complete_receiving_line")
		return
	}
	persisted, found := findReceivingItemForRequest(task, body)
	if !found {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingLine", "Línea no encontrada en la tarea", "mobile_complete_receiving_line_unknown")
		return
	}
	expectedQty := persisted.ExpectedQuantity
	if expectedQty > 0 && body.PickedQty > expectedQty*mobileReceivingOverTolerance {
		tools.ResponseBadRequest(
			ctx,
			"MobileCompleteReceivingLine",
			"received_qty excede la tolerancia permitida (5% sobre la cantidad esperada)",
			"mobile_complete_receiving_line_tolerance",
		)
		return
	}

	item := requests.ReceivingTaskItemRequest{
		SKU:              body.SKU,
		ExpectedQuantity: int(expectedQty),
		Location:         body.LocationScanned,
	}
	receivedInt := int(body.PickedQty)
	item.ReceivedQuantity = &receivedInt
	if body.Lot != "" {
		item.LotNumbers = []requests.CreateLotRequest{{
			LotNumber: body.Lot,
			SKU:       body.SKU,
			Quantity:  body.PickedQty,
		}}
	}
	if body.Serial != "" {
		item.SerialNumbers = []database.Serial{{SerialNumber: body.Serial, SKU: body.SKU}}
	}
	resp = c.Receiving.CompleteReceivingLine(id, body.LocationScanned, userID, item)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompleteReceivingLine", "mobile_complete_receiving_line", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileCompleteReceivingLine", "Línea completada", "mobile_complete_receiving_line", nil, false, "")
}

func (c *MobileController) CompleteReceivingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileCompleteReceivingTask", "mobile_complete_receiving_task", "ID de tarea inválido")
	if !ok {
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileCompleteReceivingTask", "Token inválido", "invalid_token")
		return
	}
	var body responses.MobileCompleteTaskRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingTask", "Cuerpo inválido", "mobile_complete_receiving_task")
		return
	}
	if strings.TrimSpace(body.LocationScanned) == "" {
		tools.ResponseBadRequest(ctx, "MobileCompleteReceivingTask", "location_scanned es requerido", "mobile_complete_receiving_task")
		return
	}
	resp := c.Receiving.CompleteFullTask(id, body.LocationScanned, userID)
	if resp != nil {
		writeErrorResponse(ctx, "MobileCompleteReceivingTask", "mobile_complete_receiving_task", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileCompleteReceivingTask", "Tarea completada", "mobile_complete_receiving_task", nil, false, "")
}

// ─── Stock Transfers ─────────────────────────────────────────────────────────

func (c *MobileController) ListStockTransfers(ctx *gin.Context) {
	if c.StockTransfers == nil {
		tools.ResponseOK(ctx, "MobileListStockTransfers", "Sin traslados", "mobile_list_stock_transfers", []responses.MobileStockTransferSummary{}, false, "")
		return
	}
	statusFilter := parseCSVFilter(ctx.Query("status"))

	var single string
	if len(statusFilter) == 1 {
		for k := range statusFilter {
			single = k
		}
	}

	transfers, resp := c.StockTransfers.ListStockTransfers(single)
	if resp != nil {
		writeErrorResponse(ctx, "MobileListStockTransfers", "mobile_list_stock_transfers", resp)
		return
	}

	assignedToMe := strings.EqualFold(ctx.Query("assigned_to_me"), "true")
	// W7 N2-1: operators forced to assigned_to_me=true. See ListPickingTasks.
	token := ctx.Request.Header.Get("Authorization")
	role, _ := tools.GetRole(c.Config.JWTSecret, token)
	if !assignedToMe && tools.IsOperatorRole(role) {
		assignedToMe = true
	}
	var userID string
	if assignedToMe {
		uid, err := tools.GetUserId(c.Config.JWTSecret, token)
		if err != nil {
			tools.ResponseUnauthorized(ctx, "MobileListStockTransfers", "Token inválido", "mobile_list_stock_transfers")
			return
		}
		userID = uid
	}

	out := make([]responses.MobileStockTransferSummary, 0, len(transfers))
	for _, t := range transfers {
		if assignedToMe && (t.AssignedTo == nil || *t.AssignedTo != userID) {
			continue
		}
		if len(statusFilter) > 1 && !statusFilter[strings.ToLower(t.Status)] {
			continue
		}
		out = append(out, responses.MobileStockTransferSummary{
			ID:             t.ID,
			TransferNumber: t.TransferNumber,
			Status:         t.Status,
			FromLocationID: t.FromLocationID,
			ToLocationID:   t.ToLocationID,
			AssignedTo:     t.AssignedTo,
			CreatedAt:      t.CreatedAt,
			CompletedAt:    t.CompletedAt,
		})
	}
	tools.ResponseOK(ctx, "MobileListStockTransfers", "Traslados obtenidos", "mobile_list_stock_transfers", out, false, "")
}

func (c *MobileController) GetStockTransfer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileGetStockTransfer", "mobile_get_stock_transfer", "ID inválido")
	if !ok {
		return
	}
	if c.StockTransfers == nil {
		tools.ResponseNotFound(ctx, "MobileGetStockTransfer", "Servicio no disponible", "mobile_get_stock_transfer")
		return
	}
	tr, resp := c.StockTransfers.GetStockTransferByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "MobileGetStockTransfer", "mobile_get_stock_transfer", resp)
		return
	}
	if tr == nil {
		tools.ResponseNotFound(ctx, "MobileGetStockTransfer", "Traslado no encontrado", "mobile_get_stock_transfer")
		return
	}
	lines, _ := c.StockTransfers.ListStockTransferLines(id)
	tools.ResponseOK(ctx, "MobileGetStockTransfer", "Traslado obtenido", "mobile_get_stock_transfer", gin.H{"transfer": tr, "lines": lines}, false, "")
}

func (c *MobileController) ExecuteStockTransfer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileExecuteStockTransfer", "mobile_execute_stock_transfer", "ID inválido")
	if !ok {
		return
	}
	if c.StockTransfers == nil {
		tools.ResponseInternal(ctx, "MobileExecuteStockTransfer", "Servicio no disponible", "mobile_execute_stock_transfer")
		return
	}
	token := ctx.Request.Header.Get("Authorization")
	userID, err := tools.GetUserId(c.Config.JWTSecret, token)
	if err != nil {
		tools.ResponseUnauthorized(ctx, "MobileExecuteStockTransfer", "Token inválido", "invalid_token")
		return
	}
	tr, resp := c.StockTransfers.ExecuteTransfer(id, userID)
	if resp != nil {
		writeErrorResponse(ctx, "MobileExecuteStockTransfer", "mobile_execute_stock_transfer", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileExecuteStockTransfer", "Traslado ejecutado", "mobile_execute_stock_transfer", tr, false, "")
}

// ─── Inventory ───────────────────────────────────────────────────────────────

// QueryInventory accepts ?sku=, ?barcode=, or ?location=. With sku and location → exact SKU lookup
// at that location; with sku only → all locations for that SKU; with location only → all SKUs at
// that location. Reuses GetAllInventory + filtering for simplicity; for high-cardinality data,
// the underlying repo can later expose dedicated query methods.
func (c *MobileController) QueryInventory(ctx *gin.Context) {
	sku := strings.TrimSpace(ctx.Query("sku"))
	barcode := strings.TrimSpace(ctx.Query("barcode"))
	location := strings.TrimSpace(ctx.Query("location"))

	// Currently barcode is treated as SKU lookup (matches repository.ResolveSKUByBarcode placeholder).
	if sku == "" && barcode != "" {
		sku = barcode
	}

	if sku != "" && location != "" {
		item, resp := c.Inventory.GetInventoryBySkuAndLocation(sku, location)
		if resp != nil {
			writeErrorResponse(ctx, "MobileQueryInventory", "mobile_query_inventory", resp)
			return
		}
		if item == nil {
			tools.ResponseOK(ctx, "MobileQueryInventory", "Sin resultados", "mobile_query_inventory", []interface{}{}, false, "")
			return
		}
		tools.ResponseOK(ctx, "MobileQueryInventory", "OK", "mobile_query_inventory", []interface{}{item}, false, "")
		return
	}

	all, resp := c.Inventory.GetAllInventory()
	if resp != nil {
		writeErrorResponse(ctx, "MobileQueryInventory", "mobile_query_inventory", resp)
		return
	}
	if all == nil {
		all = nil
	}

	filtered := all[:0:0]
	for _, inv := range all {
		if sku != "" && !strings.EqualFold(inv.SKU, sku) {
			continue
		}
		if location != "" && !strings.EqualFold(inv.Location, location) {
			continue
		}
		filtered = append(filtered, inv)
	}
	tools.ResponseOK(ctx, "MobileQueryInventory", "OK", "mobile_query_inventory", filtered, false, "")
}

// GetLotsBySKU returns all lots for a SKU. Reuses GetInventoryLots indirectly is awkward
// (it expects an inventoryID), so for mobile we walk the inventory list and aggregate lots.
// For the volumes mobile sees this is fine; later the repo can expose a direct method.
func (c *MobileController) GetLotsBySKU(ctx *gin.Context) {
	sku := strings.TrimSpace(ctx.Param("sku"))
	if sku == "" {
		tools.ResponseBadRequest(ctx, "MobileGetLotsBySKU", "SKU requerido", "mobile_get_lots_by_sku")
		return
	}
	all, resp := c.Inventory.GetAllInventory()
	if resp != nil {
		writeErrorResponse(ctx, "MobileGetLotsBySKU", "mobile_get_lots_by_sku", resp)
		return
	}
	out := []interface{}{}
	for _, inv := range all {
		if !strings.EqualFold(inv.SKU, sku) {
			continue
		}
		lots, lresp := c.Inventory.GetInventoryLots(inv.ID)
		if lresp != nil {
			continue
		}
		for _, l := range lots {
			out = append(out, gin.H{
				"location": inv.Location,
				"lot":      l,
			})
		}
	}
	tools.ResponseOK(ctx, "MobileGetLotsBySKU", "OK", "mobile_get_lots_by_sku", out, false, "")
}

// GetMovementsBySKU mirrors /inventory_movements/:sku but with optional ?limit query.
func (c *MobileController) GetMovementsBySKU(ctx *gin.Context) {
	sku := strings.TrimSpace(ctx.Param("sku"))
	if sku == "" {
		tools.ResponseBadRequest(ctx, "MobileGetMovementsBySKU", "SKU requerido", "mobile_get_movements_by_sku")
		return
	}
	movs, resp := c.InventoryMovements.GetAllInventoryMovements(sku)
	if resp != nil {
		writeErrorResponse(ctx, "MobileGetMovementsBySKU", "mobile_get_movements_by_sku", resp)
		return
	}
	limit := 20
	if ls := ctx.Query("limit"); ls != "" {
		if parsed, err := strconv.Atoi(ls); err == nil && parsed > 0 && parsed < 500 {
			limit = parsed
		}
	}
	if len(movs) > limit {
		movs = movs[:limit]
	}
	tools.ResponseOK(ctx, "MobileGetMovementsBySKU", "OK", "mobile_get_movements_by_sku", movs, false, "")
}

// ─── Stock Alerts ────────────────────────────────────────────────────────────

func (c *MobileController) ListStockAlerts(ctx *gin.Context) {
	if c.StockAlerts == nil {
		tools.ResponseOK(ctx, "MobileListStockAlerts", "Sin alertas", "mobile_list_stock_alerts", []interface{}{}, false, "")
		return
	}
	resolved := strings.EqualFold(ctx.Query("resolved"), "true")
	alerts, resp := c.StockAlerts.GetAllStockAlerts(resolved)
	if resp != nil {
		writeErrorResponse(ctx, "MobileListStockAlerts", "mobile_list_stock_alerts", resp)
		return
	}
	tools.ResponseOK(ctx, "MobileListStockAlerts", "OK", "mobile_list_stock_alerts", alerts, false, "")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func parseCSVFilter(raw string) map[string]bool {
	out := map[string]bool{}
	if raw == "" {
		return out
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = true
		}
	}
	return out
}

