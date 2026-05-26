package responses

// ValuationBreakdownItem is a single row in a valuation breakdown.
//
// JSON field names match the frontend contract (InventoryValuation model + the
// dashboard valuation widget): `id` (the article SKU / location / category key,
// used by the widget's `['/articles', item.id]` link), `total_value` and
// `quantity`. They previously shipped as `key`/`value`/`qty`, so the widget read
// undefined → every top item showed ₡0 and linked to /articles/undefined.
type ValuationBreakdownItem struct {
	Key   string  `json:"id"`
	Label string  `json:"label"`
	Value float64 `json:"total_value"`
	Qty   float64 `json:"quantity"`
}

// InventoryValuationResponse is the response for GET /api/inventory/valuation.
type InventoryValuationResponse struct {
	TotalValue float64                  `json:"total_value"`
	Currency   string                   `json:"currency"`
	GroupBy    string                   `json:"group_by"`
	Breakdown  []ValuationBreakdownItem `json:"breakdown"`
}
