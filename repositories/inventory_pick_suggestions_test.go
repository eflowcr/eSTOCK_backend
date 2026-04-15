package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ptr helpers
func strPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func TestAllocatePickRows_EmptyRows(t *testing.T) {
	resp := allocatePickRows(nil, 10)
	assert.Empty(t, resp.Allocations)
	assert.Equal(t, 0.0, resp.TotalFound)
	assert.False(t, resp.Sufficient)
}

func TestAllocatePickRows_ZeroQty_ReturnsAll(t *testing.T) {
	rows := []pickRow{
		{Location: "A-01", InvQty: 20, InvReserved: 0, LotQtyInLoc: 0},
		{Location: "B-01", InvQty: 15, InvReserved: 5, LotQtyInLoc: 0},
	}
	resp := allocatePickRows(rows, 0)
	// With qty=0 we return everything available
	assert.Equal(t, 30.0, resp.TotalFound) // 20 + 10
	assert.True(t, resp.Sufficient)
	assert.Len(t, resp.Allocations, 2)
}

func TestAllocatePickRows_ExactFit_SingleLocation(t *testing.T) {
	rows := []pickRow{
		{Location: "A-01", InvQty: 50, InvReserved: 10, LotQtyInLoc: 0},
	}
	resp := allocatePickRows(rows, 40)
	require.Len(t, resp.Allocations, 1)
	assert.Equal(t, 40.0, resp.Allocations[0].Quantity)
	assert.Equal(t, 40.0, resp.TotalFound)
	assert.True(t, resp.Sufficient)
}

func TestAllocatePickRows_CrossLocation_FEFO(t *testing.T) {
	exp1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // expires first
	exp2 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	lot1 := "LOT-A"
	lot2 := "LOT-B"

	// rows arrive pre-sorted (FEFO): exp1 row first
	rows := []pickRow{
		{Location: "A-01", InvQty: 10, InvReserved: 0, LotNumber: &lot1, ExpirationDate: timePtr(exp1), LotQtyInLoc: 10},
		{Location: "B-01", InvQty: 20, InvReserved: 0, LotNumber: &lot2, ExpirationDate: timePtr(exp2), LotQtyInLoc: 20},
	}
	resp := allocatePickRows(rows, 25)
	require.Len(t, resp.Allocations, 2)
	// First allocation should come from LOT-A (earlier expiry)
	assert.Equal(t, "A-01", resp.Allocations[0].Location)
	assert.Equal(t, 10.0, resp.Allocations[0].Quantity)
	assert.Equal(t, "LOT-A", *resp.Allocations[0].LotNumber)
	assert.Equal(t, "2026-03-01", *resp.Allocations[0].ExpirationDate)
	// Second allocation fills remaining 15 from LOT-B
	assert.Equal(t, "B-01", resp.Allocations[1].Location)
	assert.Equal(t, 15.0, resp.Allocations[1].Quantity)
	assert.Equal(t, 25.0, resp.TotalFound)
	assert.True(t, resp.Sufficient)
}

func TestAllocatePickRows_Insufficient(t *testing.T) {
	rows := []pickRow{
		{Location: "A-01", InvQty: 5, InvReserved: 0, LotQtyInLoc: 0},
	}
	resp := allocatePickRows(rows, 10)
	assert.Equal(t, 5.0, resp.TotalFound)
	assert.False(t, resp.Sufficient)
}

func TestAllocatePickRows_ReservedQtyReducesAvailable(t *testing.T) {
	rows := []pickRow{
		{Location: "A-01", InvQty: 20, InvReserved: 15, LotQtyInLoc: 0},
	}
	resp := allocatePickRows(rows, 10)
	// Only 5 available (20 - 15)
	assert.Equal(t, 5.0, resp.TotalFound)
	assert.False(t, resp.Sufficient)
	require.Len(t, resp.Allocations, 1)
	assert.Equal(t, 5.0, resp.Allocations[0].Quantity)
}

func TestAllocatePickRows_LotWithNoStockInLoc_Skipped(t *testing.T) {
	lot := "LOT-X"
	rows := []pickRow{
		// LotNumber set but LotQtyInLoc = 0 → should be skipped
		{Location: "A-01", InvQty: 20, InvReserved: 0, LotNumber: &lot, LotQtyInLoc: 0},
		{Location: "B-01", InvQty: 10, InvReserved: 0, LotQtyInLoc: 0}, // no lot
	}
	resp := allocatePickRows(rows, 10)
	// Row 0 skipped (lot with 0 qty). Row 1 provides 10.
	require.Len(t, resp.Allocations, 1)
	assert.Equal(t, "B-01", resp.Allocations[0].Location)
	assert.Equal(t, 10.0, resp.TotalFound)
	assert.True(t, resp.Sufficient)
}

func TestAllocatePickRows_LotQtyCapsBound(t *testing.T) {
	lot := "LOT-Y"
	rows := []pickRow{
		// location has 30 available but this specific lot only has 8 in it
		{Location: "A-01", InvQty: 30, InvReserved: 0, LotNumber: &lot, LotQtyInLoc: 8},
	}
	resp := allocatePickRows(rows, 20)
	// Should only take 8 (lot bound), not 20 or 30
	assert.Equal(t, 8.0, resp.TotalFound)
	assert.False(t, resp.Sufficient)
}

func TestAllocatePickRows_MultipleLotsInSameLocation(t *testing.T) {
	lot1, lot2 := "LOT-1", "LOT-2"
	rows := []pickRow{
		{Location: "A-01", InvQty: 30, InvReserved: 0, LotNumber: &lot1, LotQtyInLoc: 10},
		{Location: "A-01", InvQty: 30, InvReserved: 0, LotNumber: &lot2, LotQtyInLoc: 15},
	}
	resp := allocatePickRows(rows, 20)
	// Both lots from same location: 10 + 10 = 20 (second lot capped at location remaining = 20)
	assert.Equal(t, 20.0, resp.TotalFound)
	assert.True(t, resp.Sufficient)
	require.Len(t, resp.Allocations, 2)
	assert.Equal(t, 10.0, resp.Allocations[0].Quantity) // full lot-1
	assert.Equal(t, 10.0, resp.Allocations[1].Quantity) // remaining from lot-2
}
