package database

import "time"

// StockTransfer represents a WMS transfer order (from location to location).
type StockTransfer struct {
	ID             string     `json:"id"`
	TransferNumber string     `json:"transfer_number"`
	FromLocationID string     `json:"from_location_id"`
	ToLocationID   string     `json:"to_location_id"`
	Status         string     `json:"status"`
	CreatedBy      string     `json:"created_by"`
	AssignedTo     *string    `json:"assigned_to,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	DockLocation   *string    `json:"dock_location,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// StockTransferLine represents a line on a stock transfer (SKU + quantity).
type StockTransferLine struct {
	ID              string    `json:"id"`
	StockTransferID string    `json:"stock_transfer_id"`
	Sku             string    `json:"sku"`
	Quantity        float64   `json:"quantity"`
	Presentation     *string   `json:"presentation,omitempty"`
	LineStatus      string    `json:"line_status"`
	CreatedAt       time.Time `json:"created_at"`
}
