// Integration tests for Wave 4 (B3/H5) lazy reservations mechanics.
// Requires Docker (testcontainers). Run: go test -v ./repositories/... -run TestPickingB3
//
// All tests share setupGORMTestDB (defined in receiving_tasks_upsert_lot_integration_test.go)
// which spins up a Postgres 16 container and runs all migrations.

package repositories

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func newPickingRepo(db *gorm.DB) *PickingTaskRepository {
	return &PickingTaskRepository{DB: db}
}

// seedInventory inserts a single inventory row and returns its id.
func seedInventory(t *testing.T, db *gorm.DB, sku, location string, qty, reserved float64) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO inventory (id, sku, location, quantity, reserved_qty, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', NOW(), NOW())`,
		id, sku, location, qty, reserved).Error)
	return id
}

// seedArticle inserts a minimal article row so FK constraints are satisfied.
func seedArticle(t *testing.T, db *gorm.DB, sku string) {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	db.Exec(`
		INSERT INTO articles (id, sku, name, track_by_lot, track_by_serial, created_at, updated_at)
		VALUES (?, ?, ?, false, false, NOW(), NOW())
		ON CONFLICT (sku) DO NOTHING`, id, sku, "Test Article "+sku)
}

// seedUser inserts a minimal user row.
func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES (?, 'Test', 'User', ?, 'hashed', NOW(), NOW())
		ON CONFLICT (email) DO NOTHING`, id, id+"@test.com")
	// Return actual ID (may differ if ON CONFLICT skipped insert)
	var uid string
	db.Raw("SELECT id FROM users WHERE email = ?", id+"@test.com").Scan(&uid)
	if uid == "" {
		uid = id
	}
	return uid
}

// seedPickingTask creates a picking_task and returns its id.
func seedPickingTask(t *testing.T, db *gorm.DB, userID, status string, items interface{}) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	itemsJSON, _ := json.Marshal(items)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks (id, task_id, order_number, created_by, status, priority, items, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'normal', ?, NOW(), NOW())`,
		id, "PICK-"+id[:6], "ORD-"+id[:6], userID, status, string(itemsJSON)).Error)
	return id
}

// getInventory reads inventory row by id.
func getInventory(t *testing.T, db *gorm.DB, id string) database.Inventory {
	t.Helper()
	var inv database.Inventory
	require.NoError(t, db.Where("id = ?", id).First(&inv).Error)
	return inv
}

func getInventoryBySKULoc(t *testing.T, db *gorm.DB, sku, location string) database.Inventory {
	t.Helper()
	var inv database.Inventory
	require.NoError(t, db.Where("sku = ? AND location = ?", sku, location).First(&inv).Error)
	return inv
}

func getPickingTask(t *testing.T, db *gorm.DB, id string) database.PickingTask {
	t.Helper()
	var task database.PickingTask
	require.NoError(t, db.Where("id = ?", id).First(&task).Error)
	return task
}

// ─────────────────────────────────────────────────────────────────────────────
// B3a — StartPickingTask
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_StartPickingTask_HappyPath(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-A")
	invID := seedInventory(t, db, "SKU-A", "LOC-1", 100, 0)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-A", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 30},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "open", items)

	repo := newPickingRepo(db)
	resp := repo.StartPickingTask(context.Background(), taskID, userID)
	assert.Nil(t, resp, "StartPickingTask should succeed")

	// inventory.reserved_qty must increase by 30.
	inv := getInventory(t, db, invID)
	assert.Equal(t, float64(30), inv.ReservedQty)

	// task status must be in_progress.
	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "in_progress", task.Status)
}

func TestPickingB3_StartPickingTask_InsufficientStock(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-B")
	invID := seedInventory(t, db, "SKU-B", "LOC-1", 10, 0) // only 10 available

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-B", ExpectedQuantity: 20, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 20}, // requesting more than available
		}},
	}
	taskID := seedPickingTask(t, db, userID, "open", items)

	repo := newPickingRepo(db)
	resp := repo.StartPickingTask(context.Background(), taskID, userID)

	require.NotNil(t, resp, "should return an error response")
	assert.True(t, resp.Handled, "should be a handled (business) error")

	// inventory must NOT have been modified.
	inv := getInventory(t, db, invID)
	assert.Equal(t, float64(0), inv.ReservedQty)

	// task status must remain open.
	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "open", task.Status)
}

func TestPickingB3_StartPickingTask_InvalidTransition(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-C")
	seedInventory(t, db, "SKU-C", "LOC-1", 100, 0)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-C", ExpectedQuantity: 5, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 5},
		}},
	}
	// Already completed — cannot start.
	taskID := seedPickingTask(t, db, userID, "completed", items)

	repo := newPickingRepo(db)
	resp := repo.StartPickingTask(context.Background(), taskID, userID)

	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

// ─────────────────────────────────────────────────────────────────────────────
// B3b/B3c — UpdatePickingTask (reserve recalc + cancel releases)
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_UpdatePickingTask_ReserveRecalc(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-D")
	invID1 := seedInventory(t, db, "SKU-D", "LOC-1", 100, 30) // 30 already reserved
	invID2 := seedInventory(t, db, "SKU-D", "LOC-2", 50, 0)

	oldItems := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 30},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", oldItems)

	// Update items to use LOC-2 instead.
	newItems := []requests.PickingTaskItemRequest{
		{SKU: "SKU-D", ExpectedQuantity: 20, Allocations: []database.LocationAllocation{
			{Location: "LOC-2", Quantity: 20},
		}},
	}
	newItemsJSON, _ := json.Marshal(newItems)

	repo := newPickingRepo(db)
	resp := repo.UpdatePickingTask(context.Background(), taskID, map[string]interface{}{
		"items": json.RawMessage(newItemsJSON),
	}, userID)
	assert.Nil(t, resp)

	// LOC-1 reserved_qty must drop back to 0.
	inv1 := getInventory(t, db, invID1)
	assert.Equal(t, float64(0), inv1.ReservedQty)

	// LOC-2 reserved_qty must increase by 20.
	inv2 := getInventory(t, db, invID2)
	assert.Equal(t, float64(20), inv2.ReservedQty)
}

func TestPickingB3_CancelPickingTask_ReleasesReservations(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-E")
	invID := seedInventory(t, db, "SKU-E", "LOC-1", 100, 40)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-E", ExpectedQuantity: 40, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 40},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	resp := repo.UpdatePickingTask(context.Background(), taskID, map[string]interface{}{
		"status": "cancelled",
	}, userID)
	assert.Nil(t, resp)

	// reserved_qty must go back to 0.
	inv := getInventory(t, db, invID)
	assert.Equal(t, float64(0), inv.ReservedQty)

	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "cancelled", task.Status)
}

func TestPickingB3_CancelPickingTask_FromOpen_NoReservationsToRelease(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-F")
	invID := seedInventory(t, db, "SKU-F", "LOC-1", 100, 0)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-F", ExpectedQuantity: 10, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 10},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "open", items)

	repo := newPickingRepo(db)
	resp := repo.UpdatePickingTask(context.Background(), taskID, map[string]interface{}{
		"status": "cancelled",
	}, userID)
	assert.Nil(t, resp)

	// reserved_qty stays 0 (lazy — nothing was ever reserved).
	inv := getInventory(t, db, invID)
	assert.Equal(t, float64(0), inv.ReservedQty)
}

func TestPickingB3_UpdatePickingTask_InvalidTransition(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-G")

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-G", ExpectedQuantity: 5, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 5},
		}},
	}
	// cancelled → in_progress must be rejected.
	taskID := seedPickingTask(t, db, userID, "cancelled", items)

	repo := newPickingRepo(db)
	resp := repo.UpdatePickingTask(context.Background(), taskID, map[string]interface{}{
		"status": "in_progress",
	}, userID)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

// ─────────────────────────────────────────────────────────────────────────────
// B3d — CompletePickingLine
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_CompletePickingLine_DecrementsAll(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-H")
	invID := seedInventory(t, db, "SKU-H", "LOC-1", 100, 30)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-H", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 30},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	incomingItem := requests.PickingTaskItemRequest{
		SKU:              "SKU-H",
		ExpectedQuantity: 30,
		Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 30},
		},
	}

	repo := newPickingRepo(db)
	resp := repo.CompletePickingLine(context.Background(), taskID, userID, incomingItem)
	assert.Nil(t, resp)

	inv := getInventory(t, db, invID)
	// quantity decremented by pickedQty (30)
	assert.Equal(t, float64(70), inv.Quantity)
	// reserved_qty decremented by alloc.Quantity (30)
	assert.Equal(t, float64(0), inv.ReservedQty)
}

func TestPickingB3_CompletePickingLine_PartialPick(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-I")
	invID := seedInventory(t, db, "SKU-I", "LOC-1", 100, 20)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-I", ExpectedQuantity: 20, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 20},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	picked := float64(15)
	incomingItem := requests.PickingTaskItemRequest{
		SKU:              "SKU-I",
		ExpectedQuantity: 20,
		Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 20, PickedQty: &picked},
		},
	}

	repo := newPickingRepo(db)
	resp := repo.CompletePickingLine(context.Background(), taskID, userID, incomingItem)
	assert.Nil(t, resp)

	inv := getInventory(t, db, invID)
	// quantity decremented by pickedQty (15)
	assert.Equal(t, float64(85), inv.Quantity)
	// reserved_qty decremented by alloc.Quantity (20)
	assert.Equal(t, float64(0), inv.ReservedQty)
}

// ─────────────────────────────────────────────────────────────────────────────
// H5 — CompletePickingTask (full task)
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_CompletePickingTask_HappyPath(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-J")
	invID1 := seedInventory(t, db, "SKU-J", "LOC-1", 100, 20)
	invID2 := seedInventory(t, db, "SKU-J", "LOC-2", 50, 10)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-J", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 20},
			{Location: "LOC-2", Quantity: 10},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	resp := repo.CompletePickingTask(context.Background(), taskID, userID)
	assert.Nil(t, resp)

	inv1 := getInventory(t, db, invID1)
	assert.Equal(t, float64(80), inv1.Quantity)
	assert.Equal(t, float64(0), inv1.ReservedQty)

	inv2 := getInventory(t, db, invID2)
	assert.Equal(t, float64(40), inv2.Quantity)
	assert.Equal(t, float64(0), inv2.ReservedQty)

	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "completed", task.Status)
}

func TestPickingB3_CompletePickingTask_WithDifferences(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-K")
	invID := seedInventory(t, db, "SKU-K", "LOC-1", 100, 30)

	picked := float64(25) // less than alloc.Quantity(30)
	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-K", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 30, PickedQty: &picked},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "in_progress", items)

	repo := newPickingRepo(db)
	resp := repo.CompletePickingTask(context.Background(), taskID, userID)
	assert.Nil(t, resp)

	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "completed_with_differences", task.Status)

	// quantity decremented by pickedQty (25), reserved by alloc.Quantity (30)
	inv := getInventory(t, db, invID)
	assert.Equal(t, float64(75), inv.Quantity)
	assert.Equal(t, float64(0), inv.ReservedQty)
}

func TestPickingB3_CompletePickingTask_NotInProgress(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-L")

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-L", ExpectedQuantity: 5, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 5},
		}},
	}
	// open — cannot complete directly.
	taskID := seedPickingTask(t, db, userID, "open", items)

	repo := newPickingRepo(db)
	resp := repo.CompletePickingTask(context.Background(), taskID, userID)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

// ─────────────────────────────────────────────────────────────────────────────
// B4 — validateNoExpiredLots
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_StartPickingTask_ExpiredLot(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-M")
	seedInventory(t, db, "SKU-M", "LOC-1", 100, 0)

	// Insert an expired lot.
	lotID, _ := tools.GenerateNanoid(db)
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	db.Exec(`
		INSERT INTO lots (id, sku, lot_number, quantity, expiration_date, status, created_at, updated_at)
		VALUES (?, 'SKU-M', 'LOT-EXPIRED', 50, ?, 'active', NOW(), NOW())`,
		lotID, yesterday)

	lotNum := "LOT-EXPIRED"
	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-M", ExpectedQuantity: 10, Allocations: []database.LocationAllocation{
			{Location: "LOC-1", Quantity: 10, LotNumber: &lotNum},
		}},
	}
	taskID := seedPickingTask(t, db, userID, "open", items)

	repo := newPickingRepo(db)
	resp := repo.StartPickingTask(context.Background(), taskID, userID)
	require.NotNil(t, resp, "should block expired lot")
	assert.True(t, resp.Handled)

	// Status must remain open.
	task := getPickingTask(t, db, taskID)
	assert.Equal(t, "open", task.Status)
}

// ─────────────────────────────────────────────────────────────────────────────
// B3e — Adjustment blocked if new_qty < reserved_qty
// ─────────────────────────────────────────────────────────────────────────────

func TestPickingB3_Adjustment_BlockedIfBelowReserved(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-N")
	seedInventory(t, db, "SKU-N", "LOC-1", 100, 40) // 40 reserved

	repo := &AdjustmentsRepository{DB: db}
	_, resp := repo.CreateAdjustment(userID, "00000000-0000-0000-0000-000000000001", requests.CreateAdjustment{
		SKU:                "SKU-N",
		Location:           "LOC-1",
		AdjustmentQuantity: -70, // would leave qty=30, but reserved=40
		Reason:             "test",
	})
	require.NotNil(t, resp, "should block adjustment below reserved")
}

func TestPickingB3_Adjustment_AllowedIfAboveReserved(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-O")
	seedInventory(t, db, "SKU-O", "LOC-1", 100, 40)

	repo := &AdjustmentsRepository{DB: db}
	_, resp := repo.CreateAdjustment(userID, "00000000-0000-0000-0000-000000000001", requests.CreateAdjustment{
		SKU:                "SKU-O",
		Location:           "LOC-1",
		AdjustmentQuantity: -50, // leaves qty=50, which is >= reserved=40
		Reason:             "test",
	})
	assert.Nil(t, resp, "adjustment that keeps qty >= reserved should be allowed")
}
