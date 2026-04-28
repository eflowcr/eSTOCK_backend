package responses

import "time"

// HealthResponse is returned by GET /api/mobile/health.
// Note: tenant column is absent in this codebase (single-tenant). Field kept for
// forward-compat with mobile client expecting tenant info — value is empty string for now.
type MobileHealthResponse struct {
	Tenant     string    `json:"tenant"`
	User       string    `json:"user"`
	UserName   string    `json:"user_name,omitempty"`
	Email      string    `json:"email,omitempty"`
	Role       string    `json:"role"`
	ServerTime time.Time `json:"server_time"`
	Version    string    `json:"version"`
}

// MobilePickingTaskSummary is a trimmed shape used in the list view.
type MobilePickingTaskSummary struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	OrderNumber   string     `json:"order_number"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	AssignedTo    *string    `json:"assigned_to,omitempty"`
	AssigneeName  *string    `json:"assignee_name,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// MobileReceivingTaskSummary mirrors MobilePickingTaskSummary for receiving tasks.
type MobileReceivingTaskSummary struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	InboundNumber string     `json:"inbound_number"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	AssignedTo    *string    `json:"assigned_to,omitempty"`
	AssigneeName  *string    `json:"assignee_name,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// MobileStockTransferSummary is a trimmed transfer shape for the list view.
type MobileStockTransferSummary struct {
	ID             string     `json:"id"`
	TransferNumber string     `json:"transfer_number"`
	Status         string     `json:"status"`
	FromLocationID string     `json:"from_location_id"`
	ToLocationID   string     `json:"to_location_id"`
	AssignedTo     *string    `json:"assigned_to,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// MobileCompleteLineRequest is the JSON body shape for mobile complete-line endpoints
// (picking & receiving). Mobile-only: backend reads location from body so the client
// scans the location at confirmation time instead of passing it in the URL.
//
// W0.7: LineID is the deterministic line identifier emitted by the
// MobilePickingTaskDetailDto.GET response (see MobilePickingLineDto). When
// non-empty the backend looks up the matching item in the task to recover
// the real ExpectedQty for tolerance validation. When empty (older clients),
// the backend falls back to matching by (sku, lot, serial) tuple.
type MobileCompleteLineRequest struct {
	LineID          string  `json:"line_id,omitempty"`
	SKU             string  `json:"sku"`
	PickedQty       float64 `json:"picked_qty"`
	LocationScanned string  `json:"location_scanned"`
	Lot             string  `json:"lot,omitempty"`
	Serial          string  `json:"serial,omitempty"`
}

// MobileCompleteTaskRequest is the JSON body for mobile complete-full-task endpoints.
//
// W0.7: location_scanned was previously accepted but discarded server-side
// (post-W0.6 PickingTaskService.CompletePickingTask signature does not consume
// it). The field is now omitempty and the backend ignores it. Mobile clients
// MAY POST an empty body; field kept for forward-compat without breaking older
// clients still posting it.
type MobileCompleteTaskRequest struct {
	LocationScanned string `json:"location_scanned,omitempty"`
}

// MobilePickingTaskDetailDto is the GET /api/mobile/picking-tasks/:id payload.
//
// W0.7 N1-1 fix: The backend used to return database.PickingTask raw, whose
// `items` jsonb shape is []PickingTaskItem with nested allocations — mobile
// modeled `location` as a flat string per-line that never populated. This DTO
// flattens each item to a single MobilePickingLineDto with the resolved
// `location` (allocations[0].location) so the mobile contract matches the wire.
type MobilePickingTaskDetailDto struct {
	ID           string                 `json:"id"`
	TaskID       string                 `json:"task_id"`
	OrderNumber  string                 `json:"order_number"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	AssignedTo   *string                `json:"assigned_to,omitempty"`
	AssigneeName *string                `json:"assignee_name,omitempty"`
	CreatedAt    *time.Time             `json:"created_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Lines        []MobilePickingLineDto `json:"lines"`
}

// MobileReceivingTaskDetailDto is the GET /api/mobile/receiving-tasks/:id payload.
//
// W7 N1-B fix: previously the endpoint returned database.ReceivingTask raw,
// whose `items` jsonb shape is []ReceivingTaskItem with no per-line line_id and
// no flat location surface. The mobile client could not echo a stable line
// identifier back into CompleteReceivingLine, so the controller synthesized
// ExpectedQuantity = picked_qty (W0.7-equivalent picking bug). This DTO mirrors
// MobilePickingTaskDetailDto so the same line_id round-trip and tolerance
// validation pattern works for receiving.
type MobileReceivingTaskDetailDto struct {
	ID           string                   `json:"id"`
	TaskID       string                   `json:"task_id"`
	OrderNumber  string                   `json:"order_number"`
	Status       string                   `json:"status"`
	Priority     string                   `json:"priority"`
	AssignedTo   *string                  `json:"assigned_to,omitempty"`
	AssigneeName *string                  `json:"assignee_name,omitempty"`
	CreatedAt    *time.Time               `json:"created_at,omitempty"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty"`
	Lines        []MobileReceivingLineDto `json:"lines"`
}

// MobileReceivingLineDto mirrors MobilePickingLineDto but for receiving lines.
//
// LineID is a session-scoped deterministic SHA1[:12] hash of (sku|lot|serial|location)
// — same scheme as picking. Stable across GET → POST round-trip without
// requiring a per-line id column on the receiving_task_items jsonb.
//
// Status: "pending" (received_qty == 0), "partial" (0 < received_qty < expected),
// or "done" (received_qty >= expected).
type MobileReceivingLineDto struct {
	LineID      string  `json:"line_id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name,omitempty"`
	ExpectedQty float64 `json:"expected_qty"`
	ReceivedQty float64 `json:"received_qty"`
	Status      string  `json:"status"`
	Location    string  `json:"location"`
	Lot         string  `json:"lot,omitempty"`
	Serial      string  `json:"serial,omitempty"`
}

// MobilePickingLineDto is the per-line shape under MobilePickingTaskDetailDto.
//
// LineID is a deterministic identifier for a line within a task: it is a
// short hash of (sku|lot|serial|location) and is stable across GET → POST
// round-trips for the same task (so the mobile client can echo it back in
// CompletePickingLine and the backend recovers ExpectedQty for tolerance
// validation). It is NOT persisted — it's recomputed each request.
//
// Location is resolved from allocations[0].location. Multi-allocation
// (split picking) currently surfaces only the first allocation; full
// multi-alloc support is TODO (see GetPickingTask in mobile_controller.go).
//
// PickedQty is the sum across the item's allocations (handles single-alloc
// trivially, and the future multi-alloc case correctly).
//
// Status: "pending" (picked_qty == 0), "partial" (0 < picked_qty < expected),
// or "done" (picked_qty >= expected).
type MobilePickingLineDto struct {
	LineID      string  `json:"line_id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name,omitempty"`
	ExpectedQty float64 `json:"expected_qty"`
	PickedQty   float64 `json:"picked_qty"`
	Status      string  `json:"status"`
	Location    string  `json:"location"`
	Lot         string  `json:"lot,omitempty"`
	Serial      string  `json:"serial,omitempty"`
}
