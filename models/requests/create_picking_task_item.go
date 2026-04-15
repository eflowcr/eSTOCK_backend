package requests

import (
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/database"
)

// PickingTaskItemRequest is the request body for a single picking task item.
// With S1 A1 (cross-location), picking is described by Allocations (list of
// location+qty pairs) instead of a single Location field.
type PickingTaskItemRequest struct {
	SKU              string                        `json:"sku" validate:"required"`
	ExpectedQuantity float64                       `json:"required_qty" validate:"required,gt=0"`
	Allocations      []database.LocationAllocation `json:"allocations" validate:"required,min=1,dive"`
	LotNumbers       []database.LotEntry           `json:"lots,omitempty"`
	SerialNumbers    []database.Serial             `json:"serials,omitempty"`
	Status           *string                       `json:"status,omitempty"`
	PickedQty        *float64                      `json:"picked_qty,omitempty"`
}

// CreatePickingTaskItemRequest is an alias kept for backwards compatibility with
// callers that reference the old type name.
type CreatePickingTaskItemRequest = PickingTaskItemRequest

// ValidateAllocationSum confirms that the sum of all allocations equals
// ExpectedQuantity (tolerance 0.001 for floating-point arithmetic).
// The frontend validates this too (F2c) but the backend re-validates for
// direct API callers.
func (r PickingTaskItemRequest) ValidateAllocationSum() error {
	sum := 0.0
	for _, a := range r.Allocations {
		sum += a.Quantity
	}
	diff := sum - r.ExpectedQuantity
	if diff > 0.001 || diff < -0.001 {
		return fmt.Errorf(
			"suma de allocations (%.3f) no coincide con required_qty (%.3f) para SKU %s",
			sum, r.ExpectedQuantity, r.SKU,
		)
	}
	return nil
}
