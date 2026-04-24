// Integration tests for D2 — OUTBOUND inventory_movements on picking completion.
// Requires Docker (testcontainers). Run: go test -v ./repositories/... -run TestPickingD2
package repositories

import (
	"context"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// D2 — CompletePickingLine → 1 OUTBOUND movement
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingD2_CompletePickingLine_EmitsOutboundMovement(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-D2A")
	seedInventory(t, db, "SKU-D2A", "LOC-1", 100, 30) // 100 qty, 30 reserved

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D2A", ExpectedQuantity: 10, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 10},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	alloc := requests.PickingTaskItemRequest{
		SKU: "SKU-D2A",
		Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 10},
		},
	}
	resp := repo.CompletePickingLine(context.Background(), taskID, userID, alloc)
	require.Nil(t, resp, "CompletePickingLine should succeed")

	// Expect exactly 1 OUTBOUND movement for this task.
	var movements []database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ? AND movement_type = 'outbound'", taskID).Find(&movements).Error)
	require.Len(t, movements, 1, "should have 1 outbound movement")

	mov := movements[0]
	assert.NotEmpty(t, mov.ID, "movement id must be non-empty (nanoid)")
	assert.Equal(t, "SKU-D2A", mov.SKU)
	assert.Equal(t, "LOC-1", mov.Location)
	assert.Equal(t, float64(10), mov.Quantity)
	assert.Equal(t, "outbound", mov.MovementType)
	require.NotNil(t, mov.ReferenceType)
	assert.Equal(t, "picking_task", *mov.ReferenceType)
	require.NotNil(t, mov.ReferenceID)
	assert.Equal(t, taskID, *mov.ReferenceID)
	require.NotNil(t, mov.BeforeQty)
	assert.Equal(t, float64(100), *mov.BeforeQty)
	require.NotNil(t, mov.AfterQty)
	assert.Equal(t, float64(90), *mov.AfterQty)
	require.NotNil(t, mov.UserID)
	assert.Equal(t, userID, *mov.UserID)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 — CompletePickingTask with 2 allocations → 2 OUTBOUND movements
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingD2_CompletePickingTask_TwoAllocations_TwoMovements(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-D2B")
	seedInventory(t, db, "SKU-D2B", "LOC-A", 50, 20)
	seedInventory(t, db, "SKU-D2B", "LOC-B", 30, 10)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D2B", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-A", Quantity: 20},
			{Location: "LOC-B", Quantity: 10},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	resp := repo.CompletePickingTask(context.Background(), taskID, userID)
	require.Nil(t, resp, "CompletePickingTask should succeed")

	// Expect 2 OUTBOUND movements (one per allocation).
	var movements []database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ? AND movement_type = 'outbound'", taskID).
		Order("location").Find(&movements).Error)
	require.Len(t, movements, 2, "should have 2 outbound movements")

	for _, mov := range movements {
		assert.NotEmpty(t, mov.ID)
		require.NotNil(t, mov.ReferenceType)
		assert.Equal(t, "picking_task", *mov.ReferenceType)
		require.NotNil(t, mov.ReferenceID)
		assert.Equal(t, taskID, *mov.ReferenceID)
		require.NotNil(t, mov.BeforeQty)
		require.NotNil(t, mov.AfterQty)
		require.NotNil(t, mov.UserID)
		assert.Equal(t, userID, *mov.UserID)
	}

	// LOC-A: started at 50, picked 20 → after 30.
	var movA database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ? AND location = 'LOC-A'", taskID).First(&movA).Error)
	assert.Equal(t, float64(50), *movA.BeforeQty)
	assert.Equal(t, float64(30), *movA.AfterQty)

	// LOC-B: started at 30, picked 10 → after 20.
	var movB database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ? AND location = 'LOC-B'", taskID).First(&movB).Error)
	assert.Equal(t, float64(30), *movB.BeforeQty)
	assert.Equal(t, float64(20), *movB.AfterQty)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 — Sum of movement qty == total picked qty
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingD2_MovementQuantityMatchesPicked(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-D2C")
	seedInventory(t, db, "SKU-D2C", "LOC-X", 200, 0)

	pickedQty := 25.0
	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D2C", ExpectedQuantity: pickedQty, Allocations: []database.LocationAllocation{
			{Location: "LOC-X", Quantity: pickedQty},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	resp := repo.CompletePickingTask(context.Background(), taskID, userID)
	require.Nil(t, resp)

	var movements []database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ?", taskID).Find(&movements).Error)
	var total float64
	for _, m := range movements {
		total += m.Quantity
	}
	assert.Equal(t, pickedQty, total)

	inv := getInventoryBySKULoc(t, db, "SKU-D2C", "LOC-X")
	assert.Equal(t, 200-pickedQty, inv.Quantity)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 — CompletePickingLine with lot resolves lot_id
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingD2_CompletePickingLine_WithLot_PopulatesLotID(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-D2LOT")
	seedInventory(t, db, "SKU-D2LOT", "LOC-L", 50, 10)

	var lotID string
	require.NoError(t, db.Raw("SELECT nanoid()").Scan(&lotID).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO lots (id, lot_number, sku, quantity, status, created_at, updated_at)
		VALUES (?, 'LOT-001', 'SKU-D2LOT', 50, 'available', NOW(), NOW())`, lotID).Error)

	// inventory_lot entry so the lot decrement works.
	var ilID string
	require.NoError(t, db.Raw("SELECT nanoid()").Scan(&ilID).Error)
	var invID string
	require.NoError(t, db.Raw("SELECT id FROM inventory WHERE sku = 'SKU-D2LOT' AND location = 'LOC-L'").Scan(&invID).Error)
	// S3.5 W2-A: tenant_id NOT NULL after migration 000034 — backfill to default tenant.
	db.Exec(`INSERT INTO inventory_lots (id, inventory_id, lot_id, quantity, location, created_at, tenant_id) VALUES (?, ?, ?, 50, 'LOC-L', NOW(), '00000000-0000-0000-0000-000000000001'::uuid)`, ilID, invID, lotID)

	lotNum := "LOT-001"
	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D2LOT", ExpectedQuantity: 5, Allocations: []database.LocationAllocation{
			{Location: "LOC-L", Quantity: 5, LotNumber: &lotNum},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	alloc := requests.PickingTaskItemRequest{
		SKU: "SKU-D2LOT",
		Allocations: []database.LocationAllocation{
			{Location: "LOC-L", Quantity: 5, LotNumber: &lotNum},
		},
	}
	resp := repo.CompletePickingLine(context.Background(), taskID, userID, alloc)
	require.Nil(t, resp)

	var mov database.InventoryMovement
	require.NoError(t, db.Where("reference_id = ? AND movement_type = 'outbound'", taskID).First(&mov).Error)
	require.NotNil(t, mov.LotID, "lot_id must be populated")
	assert.Equal(t, lotID, *mov.LotID)
}
