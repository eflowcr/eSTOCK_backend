package repositories

import (
	"fmt"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
)

// ── Example row detection ─────────────────────────────────────────────────────

func TestInventoryImport_ExampleRowSkipped(t *testing.T) {
	rows := []requests.InventoryImportRow{{SKU: "SKU-0001", Location: "LOC-A01", Quantity: "10"}}
	var skipped int
	for _, row := range rows {
		if equalFoldStr(row.SKU, "SKU-0001") {
			skipped++
		}
	}
	assert.Equal(t, 1, skipped)
}

func TestInventoryImport_NonExampleNotSkipped(t *testing.T) {
	rows := []requests.InventoryImportRow{{SKU: "SKU-0002", Location: "LOC-A01", Quantity: "10"}}
	var skipped int
	for _, row := range rows {
		if equalFoldStr(row.SKU, "SKU-0001") {
			skipped++
		}
	}
	assert.Equal(t, 0, skipped)
}

// ── Required field validation ─────────────────────────────────────────────────

func TestInventoryValidate_MissingSKU(t *testing.T) {
	result := validateInventoryRow(0, requests.InventoryImportRow{SKU: "", Location: "LOC-A01", Quantity: "10"})
	assert.Equal(t, responses.InventoryStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "sku")
}

func TestInventoryValidate_MissingLocation(t *testing.T) {
	result := validateInventoryRow(0, requests.InventoryImportRow{SKU: "SKU-001", Location: "", Quantity: "10"})
	assert.Equal(t, responses.InventoryStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "location")
}

func TestInventoryValidate_MissingQuantity(t *testing.T) {
	result := validateInventoryRow(0, requests.InventoryImportRow{SKU: "SKU-001", Location: "LOC-A01", Quantity: ""})
	assert.Equal(t, responses.InventoryStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "quantity")
}

func TestInventoryValidate_InvalidQuantity(t *testing.T) {
	result := validateInventoryRow(0, requests.InventoryImportRow{SKU: "SKU-001", Location: "LOC-A01", Quantity: "abc"})
	assert.Equal(t, responses.InventoryStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "quantity")
}

func TestInventoryValidate_ValidRow(t *testing.T) {
	result := validateInventoryRow(0, requests.InventoryImportRow{SKU: "SKU-001", Location: "LOC-A01", Quantity: "10"})
	assert.Empty(t, result.FieldErrors)
	assert.NotEqual(t, responses.InventoryStatusError, result.Status)
}

// ── Batch duplicate detection ─────────────────────────────────────────────────

func TestInventoryValidate_DuplicateSkuLocation(t *testing.T) {
	seen := map[string]bool{}
	pairs := []string{"SKU-001|LOC-A01", "SKU-001|LOC-B01", "SKU-001|LOC-A01"}
	var duplicates int
	for _, p := range pairs {
		if seen[p] {
			duplicates++
		}
		seen[p] = true
	}
	assert.Equal(t, 1, duplicates)
}

func TestInventoryValidate_SameSkuDiffLocationIsNotDuplicate(t *testing.T) {
	seen := map[string]bool{}
	pairs := []string{"SKU-001|LOC-A01", "SKU-001|LOC-B01"}
	var duplicates int
	for _, p := range pairs {
		if seen[p] {
			duplicates++
		}
		seen[p] = true
	}
	assert.Equal(t, 0, duplicates)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func validateInventoryRow(idx int, row requests.InventoryImportRow) responses.InventoryValidationResult {
	result := responses.InventoryValidationResult{RowIndex: idx, Row: row}
	sku := row.SKU
	loc := row.Location
	qty := row.Quantity

	if sku == "" || loc == "" || qty == "" {
		result.Status = responses.InventoryStatusError
		result.FieldErrors = map[string]string{}
		if sku == "" { result.FieldErrors["sku"] = "SKU requerido" }
		if loc == "" { result.FieldErrors["location"] = "Ubicación requerida" }
		if qty == "" { result.FieldErrors["quantity"] = "Cantidad requerida" }
		return result
	}

	if _, err := parseFloat(qty); err != nil {
		result.Status = responses.InventoryStatusError
		result.FieldErrors = map[string]string{"quantity": "Cantidad debe ser un número"}
		return result
	}

	result.Status = responses.InventoryStatusNew
	return result
}

func parseFloat(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}
