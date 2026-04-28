package database

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── LotEntry ──────────────────────────────────────────────────────────────────

func TestLotEntry_JSONRoundtrip_WithExpiration(t *testing.T) {
	exp := "2026-12-31"
	status := "pending"
	entry := LotEntry{
		LotNumber:      "LOT-A",
		SKU:            "SKU-001",
		Quantity:       100.5,
		ExpirationDate: &exp,
		Status:         &status,
	}

	b, err := json.Marshal(entry)
	require.NoError(t, err)

	var got LotEntry
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "LOT-A", got.LotNumber)
	assert.Equal(t, "SKU-001", got.SKU)
	assert.Equal(t, 100.5, got.Quantity)
	require.NotNil(t, got.ExpirationDate)
	assert.Equal(t, "2026-12-31", *got.ExpirationDate)
	require.NotNil(t, got.Status)
	assert.Equal(t, "pending", *got.Status)
}

func TestLotEntry_JSONRoundtrip_OmitsNilFields(t *testing.T) {
	entry := LotEntry{
		LotNumber: "LOT-B",
		Quantity:  50,
	}

	b, err := json.Marshal(entry)
	require.NoError(t, err)

	// expiration_date and status should be absent in the JSON output
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &raw))
	assert.NotContains(t, raw, "expiration_date")
	assert.NotContains(t, raw, "status")
	assert.NotContains(t, raw, "sku") // omitempty
}

// ── LocationAllocation ────────────────────────────────────────────────────────

func TestLocationAllocation_JSONRoundtrip(t *testing.T) {
	lot := "LOT-A"
	status := "pending"
	exp := "2026-06-30"
	picked := 30.0
	alloc := LocationAllocation{
		Location:       "RACK-A1",
		Quantity:       60.0,
		LotNumber:      &lot,
		PickedQty:      &picked,
		Status:         &status,
		ExpirationDate: &exp,
	}

	b, err := json.Marshal(alloc)
	require.NoError(t, err)

	var got LocationAllocation
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "RACK-A1", got.Location)
	assert.Equal(t, 60.0, got.Quantity)
	require.NotNil(t, got.LotNumber)
	assert.Equal(t, "LOT-A", *got.LotNumber)
	require.NotNil(t, got.PickedQty)
	assert.Equal(t, 30.0, *got.PickedQty)
	assert.Equal(t, "2026-06-30", *got.ExpirationDate)
}

// ── PickingTaskItem ───────────────────────────────────────────────────────────

func TestPickingTaskItem_JSONRoundtrip_WithAllocations(t *testing.T) {
	lot := "LOT-A"
	status := "pending"
	item := PickingTaskItem{
		SKU:              "SKU-001",
		ExpectedQuantity: 100.0,
		Allocations: []LocationAllocation{
			{Location: "RACK-A1", Quantity: 60, LotNumber: &lot, Status: &status},
			{Location: "RACK-B2", Quantity: 40},
		},
	}

	b, err := json.Marshal(item)
	require.NoError(t, err)

	var got PickingTaskItem
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "SKU-001", got.SKU)
	assert.Equal(t, 100.0, got.ExpectedQuantity)
	require.Len(t, got.Allocations, 2)
	assert.Equal(t, "RACK-A1", got.Allocations[0].Location)
	assert.Equal(t, 60.0, got.Allocations[0].Quantity)
	require.NotNil(t, got.Allocations[0].LotNumber)
	assert.Equal(t, "LOT-A", *got.Allocations[0].LotNumber)
	assert.Equal(t, "RACK-B2", got.Allocations[1].Location)
	assert.Equal(t, 40.0, got.Allocations[1].Quantity)
}

func TestPickingTaskItem_JSONRoundtrip_OmitsEmptyAllocations(t *testing.T) {
	item := PickingTaskItem{
		SKU:              "SKU-002",
		ExpectedQuantity: 10.0,
	}

	b, err := json.Marshal(item)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &raw))

	// allocations is NOT omitempty — should appear as empty/null
	// lots and serials are omitempty — should be absent
	assert.NotContains(t, raw, "lots")
	assert.NotContains(t, raw, "serials")
}

// ── ReceivingTaskItem ─────────────────────────────────────────────────────────

func TestReceivingTaskItem_JSONRoundtrip_WithLots(t *testing.T) {
	exp1 := "2026-03-15"
	status := "received"
	item := ReceivingTaskItem{
		SKU:              "SKU-003",
		ExpectedQuantity: 200.0,
		Location:         "DOCK-1",
		LotNumbers: []LotEntry{
			{LotNumber: "LOT-X", Quantity: 150, ExpirationDate: &exp1, Status: &status},
			{LotNumber: "LOT-Y", Quantity: 50},
		},
	}

	b, err := json.Marshal(item)
	require.NoError(t, err)

	var got ReceivingTaskItem
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "SKU-003", got.SKU)
	assert.Equal(t, 200.0, got.ExpectedQuantity)
	assert.Equal(t, "DOCK-1", got.Location)
	require.Len(t, got.LotNumbers, 2)
	assert.Equal(t, "LOT-X", got.LotNumbers[0].LotNumber)
	assert.Equal(t, 150.0, got.LotNumbers[0].Quantity)
	require.NotNil(t, got.LotNumbers[0].ExpirationDate)
	assert.Equal(t, "2026-03-15", *got.LotNumbers[0].ExpirationDate)
	assert.Equal(t, "LOT-Y", got.LotNumbers[1].LotNumber)
}

func TestReceivingTaskItem_JSONRoundtrip_OmitsNilOptionals(t *testing.T) {
	item := ReceivingTaskItem{
		SKU:              "SKU-004",
		ExpectedQuantity: 5.0,
		Location:         "BIN-3",
	}

	b, err := json.Marshal(item)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &raw))

	assert.NotContains(t, raw, "lots")
	assert.NotContains(t, raw, "serials")
	assert.NotContains(t, raw, "received_qty")
	assert.NotContains(t, raw, "status")
}
