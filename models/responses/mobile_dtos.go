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
type MobileCompleteLineRequest struct {
	LineID          string  `json:"line_id"`
	SKU             string  `json:"sku"`
	PickedQty       float64 `json:"picked_qty"`
	LocationScanned string  `json:"location_scanned"`
	Lot             string  `json:"lot"`
	Serial          string  `json:"serial"`
}

// MobileCompleteTaskRequest is the JSON body for mobile complete-full-task endpoints.
type MobileCompleteTaskRequest struct {
	LocationScanned string `json:"location_scanned"`
}
