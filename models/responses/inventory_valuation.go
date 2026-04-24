package responses

// ValuationBreakdownItem is a single row in a valuation breakdown.
type ValuationBreakdownItem struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Qty   float64 `json:"qty"`
}

// InventoryValuationResponse is the response for GET /api/inventory/valuation.
type InventoryValuationResponse struct {
	TotalValue float64                  `json:"total_value"`
	Currency   string                   `json:"currency"`
	GroupBy    string                   `json:"group_by"`
	Breakdown  []ValuationBreakdownItem `json:"breakdown"`
}
