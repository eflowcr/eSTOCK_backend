package requests

// StockTransferLineInput is a line for create/update (SKU, quantity, optional presentation).
type StockTransferLineInput struct {
	Sku         string   `json:"sku" binding:"required"`
	Quantity    float64  `json:"quantity" binding:"required" validate:"required,gt=0"`
	Presentation *string `json:"presentation"`
}

// StockTransferCreate is the request body for creating a stock transfer (header + lines).
type StockTransferCreate struct {
	FromLocationID string                   `json:"from_location_id" binding:"required"`
	ToLocationID   string                   `json:"to_location_id" binding:"required"`
	AssignedTo     *string                  `json:"assigned_to"`
	Notes          *string                  `json:"notes"`
	DockLocation   *string                  `json:"dock_location"`
	Lines          []StockTransferLineInput `json:"lines" binding:"required,dive"`
}

// StockTransferUpdate is the request body for updating a stock transfer header.
type StockTransferUpdate struct {
	FromLocationID string  `json:"from_location_id" binding:"required"`
	ToLocationID   string  `json:"to_location_id" binding:"required"`
	Status         string  `json:"status" binding:"required" validate:"required,oneof=draft in_progress completed cancelled"`
	AssignedTo     *string `json:"assigned_to"`
	Notes          *string `json:"notes"`
	DockLocation   *string `json:"dock_location"`
}

// StockTransferLineUpdate is the request body for updating a transfer line.
type StockTransferLineUpdate struct {
	Quantity    float64  `json:"quantity" validate:"gt=0"`
	Presentation *string `json:"presentation"`
	LineStatus  string   `json:"line_status" validate:"omitempty,oneof=pending picked received cancelled"`
}
