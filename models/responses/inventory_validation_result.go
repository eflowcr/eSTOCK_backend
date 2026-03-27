package responses

import "github.com/eflowcr/eSTOCK_backend/models/requests"

// InventoryValidationStatus describes the result of validating one import row.
type InventoryValidationStatus string

const (
	InventoryStatusNew       InventoryValidationStatus = "new"
	InventoryStatusExists    InventoryValidationStatus = "exists"    // same SKU+location in DB
	InventoryStatusSimilar   InventoryValidationStatus = "similar"   // same SKU, different location
	InventoryStatusError     InventoryValidationStatus = "error"
	InventoryStatusDuplicate InventoryValidationStatus = "duplicate" // same SKU+location in batch
)

// InventoryValidationMatch is a compact representation of an existing inventory item.
type InventoryValidationMatch struct {
	ID       string  `json:"id"`
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Location string  `json:"location"`
	Quantity float64 `json:"quantity"`
}

// InventoryValidationResult is the per-row output of the validate endpoint.
type InventoryValidationResult struct {
	RowIndex          int                          `json:"row_index"`
	Status            InventoryValidationStatus    `json:"status"`
	Row               requests.InventoryImportRow  `json:"row"`
	FieldErrors       map[string]string            `json:"field_errors,omitempty"`
	ExistingInventory *InventoryValidationMatch    `json:"existing_inventory,omitempty"`
	SimilarInventory  []InventoryValidationMatch   `json:"similar_inventory,omitempty"`
}
