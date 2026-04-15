// Unit tests for Wave 4 (B3/H1/H2/H5) that do NOT require a database.
// Tests that need a DB are in picking_task_b3_integration_test.go.
package repositories

import (
	"encoding/json"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
)

// ─────────────────────────────────────────────────────────────────────────────
// H1 — ValidateAllocationSum
// ─────────────────────────────────────────────────────────────────────────────

func TestValidateAllocationSum(t *testing.T) {
	tests := []struct {
		name    string
		item    requests.PickingTaskItemRequest
		wantErr bool
	}{
		{
			name: "single allocation, exact match",
			item: requests.PickingTaskItemRequest{
				SKU:              "SKU-001",
				ExpectedQuantity: 10,
				Allocations: []database.LocationAllocation{
					{Location: "LOC-A", Quantity: 10},
				},
			},
			wantErr: false,
		},
		{
			name: "two allocations, exact match",
			item: requests.PickingTaskItemRequest{
				SKU:              "SKU-001",
				ExpectedQuantity: 15,
				Allocations: []database.LocationAllocation{
					{Location: "LOC-A", Quantity: 10},
					{Location: "LOC-B", Quantity: 5},
				},
			},
			wantErr: false,
		},
		{
			name: "within float tolerance",
			item: requests.PickingTaskItemRequest{
				SKU:              "SKU-001",
				ExpectedQuantity: 10.0,
				Allocations: []database.LocationAllocation{
					{Location: "LOC-A", Quantity: 9.9999},
				},
			},
			wantErr: false, // diff 0.0001 < tolerance 0.001
		},
		{
			name: "sum too small (over tolerance)",
			item: requests.PickingTaskItemRequest{
				SKU:              "SKU-001",
				ExpectedQuantity: 10,
				Allocations: []database.LocationAllocation{
					{Location: "LOC-A", Quantity: 8},
				},
			},
			wantErr: true,
		},
		{
			name: "sum too large (over tolerance)",
			item: requests.PickingTaskItemRequest{
				SKU:              "SKU-001",
				ExpectedQuantity: 10,
				Allocations: []database.LocationAllocation{
					{Location: "LOC-A", Quantity: 11},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.ValidateAllocationSum()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllocationSum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// H2 — parsePickingItemsWithLegacyFallback
// ─────────────────────────────────────────────────────────────────────────────

func TestParsePickingItemsWithLegacyFallback_NewFormat(t *testing.T) {
	// New format: items already have allocations — must pass through unchanged.
	raw := json.RawMessage(`[
		{
			"sku": "SKU-001",
			"required_qty": 10,
			"allocations": [
				{"location": "LOC-A", "quantity": 6},
				{"location": "LOC-B", "quantity": 4}
			]
		}
	]`)

	items, err := parsePickingItemsWithLegacyFallback(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].Allocations) != 2 {
		t.Errorf("expected 2 allocations, got %d", len(items[0].Allocations))
	}
}

func TestParsePickingItemsWithLegacyFallback_LegacyFormat(t *testing.T) {
	// Legacy format: item has "location" string but no "allocations" — must synthesize one allocation.
	raw := json.RawMessage(`[
		{
			"sku": "SKU-001",
			"required_qty": 5,
			"location": "LOC-OLD"
		}
	]`)

	items, err := parsePickingItemsWithLegacyFallback(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].Allocations) != 1 {
		t.Fatalf("expected synthetic allocation, got %d", len(items[0].Allocations))
	}
	alloc := items[0].Allocations[0]
	if alloc.Location != "LOC-OLD" {
		t.Errorf("expected location LOC-OLD, got %s", alloc.Location)
	}
	if alloc.Quantity != 5 {
		t.Errorf("expected quantity 5, got %.2f", alloc.Quantity)
	}
}

func TestParsePickingItemsWithLegacyFallback_MixedFormat(t *testing.T) {
	// First item is new format, second is legacy — both should work.
	raw := json.RawMessage(`[
		{
			"sku": "SKU-001",
			"required_qty": 10,
			"allocations": [{"location": "LOC-A", "quantity": 10}]
		},
		{
			"sku": "SKU-002",
			"required_qty": 3,
			"location": "LOC-B"
		}
	]`)

	items, err := parsePickingItemsWithLegacyFallback(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// First item untouched.
	if len(items[0].Allocations) != 1 || items[0].Allocations[0].Location != "LOC-A" {
		t.Errorf("first item allocation unexpected: %+v", items[0].Allocations)
	}
	// Second item synthesized.
	if len(items[1].Allocations) != 1 || items[1].Allocations[0].Location != "LOC-B" {
		t.Errorf("second item allocation unexpected: %+v", items[1].Allocations)
	}
}

func TestParsePickingItemsWithLegacyFallback_EmptyLocation(t *testing.T) {
	// Legacy item with empty location string — should not get a synthetic allocation.
	raw := json.RawMessage(`[{"sku": "SKU-001", "required_qty": 5, "location": ""}]`)
	items, err := parsePickingItemsWithLegacyFallback(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items[0].Allocations) != 0 {
		t.Errorf("expected no allocation for empty location, got %d", len(items[0].Allocations))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// sanitizePickingUpdatePayload
// ─────────────────────────────────────────────────────────────────────────────

func TestSanitizePickingUpdatePayload(t *testing.T) {
	data := map[string]interface{}{
		"id":         "should-be-filtered",
		"task_id":    "should-be-filtered",
		"created_at": "should-be-filtered",
		"status":     "in_progress",
		"notes":      "test",
		"unknown":    "should-be-filtered",
		"AssignedTo": "user-123",
	}

	clean := sanitizePickingUpdatePayload(data)

	if _, ok := clean["id"]; ok {
		t.Error("id should be filtered (protected)")
	}
	if _, ok := clean["task_id"]; ok {
		t.Error("task_id should be filtered (protected)")
	}
	if _, ok := clean["unknown"]; ok {
		t.Error("unknown fields should be filtered (not in whitelist)")
	}
	if clean["status"] != "in_progress" {
		t.Errorf("status should be preserved, got %v", clean["status"])
	}
	if clean["notes"] != "test" {
		t.Errorf("notes should be preserved, got %v", clean["notes"])
	}
	// AssignedTo camelCase should be normalised to assigned_to.
	if _, ok := clean["assigned_to"]; !ok {
		t.Error("AssignedTo should be normalised to assigned_to")
	}
}
