package responses

import "time"

// LotTraceSupplier is the supplier embedded in LotTraceOrigin.
type LotTraceSupplier struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// LotTraceOrigin describes where the lot came from.
type LotTraceOrigin struct {
	ReceivingTaskID string            `json:"receiving_task_id"`
	Supplier        *LotTraceSupplier `json:"supplier"`
	ReceivedAt      time.Time         `json:"received_at"`
}

// LotTraceMovement is a single movement in the lot's history.
type LotTraceMovement struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Qty           float64    `json:"qty"`
	BeforeQty     *float64   `json:"before_qty"`
	AfterQty      *float64   `json:"after_qty"`
	Location      string     `json:"location"`
	ReferenceType *string    `json:"reference_type"`
	ReferenceID   *string    `json:"reference_id"`
	UserID        *string    `json:"user_id"`
	UnitCost      *float64   `json:"unit_cost"`
	CreatedAt     time.Time  `json:"created_at"`
}

// LotTraceLocationQty is qty per location in current stock.
type LotTraceLocationQty struct {
	Location string  `json:"location"`
	Qty      float64 `json:"qty"`
}

// LotTraceCurrentStock is the current stock summary for a lot.
type LotTraceCurrentStock struct {
	TotalQty   float64               `json:"total_qty"`
	ByLocation []LotTraceLocationQty `json:"by_location"`
}

// LotTraceLot holds the core lot fields in the trace response.
type LotTraceLot struct {
	ID             string     `json:"id"`
	LotNumber      string     `json:"lot_number"`
	SKU            string     `json:"sku"`
	ExpirationDate *time.Time `json:"expiration_date"`
	ManufacturedAt *time.Time `json:"manufactured_at"`
	BestBeforeDate *time.Time `json:"best_before_date"`
	Status         *string    `json:"status"`
}

// LotTraceResponse is the full trace for GET /api/lots/:id/trace.
type LotTraceResponse struct {
	Lot          LotTraceLot           `json:"lot"`
	Origin       *LotTraceOrigin       `json:"origin"`
	Movements    []LotTraceMovement    `json:"movements"`
	CurrentStock LotTraceCurrentStock  `json:"current_stock"`
}
