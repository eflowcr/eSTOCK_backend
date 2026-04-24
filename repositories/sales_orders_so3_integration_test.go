// Integration tests for SO3 — PickingTask completion advances Sales Order status.
// Verifies that SORepository is correctly wired into PickingTaskRepository so that
// CompletePickingTask propagates picked quantities to the linked sales order.
//
// Requires Docker (testcontainers). Skipped automatically in -short mode.
// Run: go test -v ./repositories/... -run "TestSO3"

package repositories

import (
	"context"
	"encoding/json"
	"testing"

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

// seedCustomer inserts a client of type 'customer' and returns its id.
func seedCustomer(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO clients (id, tenant_id, type, code, name, is_active, created_at, updated_at)
		VALUES (?, ?, 'customer', ?, ?, true, NOW(), NOW())`,
		id, tenantID, "CUS-"+id[:6], "Customer "+id[:6]).Error)
	return id
}

// seedSalesOrder inserts a minimal sales_order and returns its id.
func seedSalesOrder(t *testing.T, db *gorm.DB, tenantID, customerID, userID string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO sales_orders (id, tenant_id, so_number, customer_id, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'submitted', ?, NOW(), NOW())`,
		id, tenantID, "SO-TEST-"+id[:6], customerID, userID).Error)
	return id
}

// seedSalesOrderItem inserts a sales_order_item and returns its id.
func seedSalesOrderItem(t *testing.T, db *gorm.DB, soID, sku string, expectedQty float64) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO sales_order_items (id, sales_order_id, article_sku, expected_qty, picked_qty, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, NOW(), NOW())`,
		id, soID, sku, expectedQty).Error)
	return id
}

// seedPickingTaskLinkedToSO inserts a picking_task with sales_order_id link.
func seedPickingTaskLinkedToSO(t *testing.T, db *gorm.DB, userID, salesOrderID, status string, items interface{}) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	itemsJSON, _ := json.Marshal(items)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks (id, task_id, order_number, created_by, status, priority, items, sales_order_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'normal', ?, ?, NOW(), NOW())`,
		id, "PICK-"+id[:6], "ORD-"+id[:6], userID, status, string(itemsJSON), salesOrderID).Error)
	return id
}

// getSalesOrderStatus reads the current status of a sales order.
func getSalesOrderStatus(t *testing.T, db *gorm.DB, soID string) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw("SELECT status FROM sales_orders WHERE id = ?", soID).Scan(&status).Error)
	return status
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3a — Complete picking → SO advances to 'completed' when fully picked
// ─────────────────────────────────────────────────────────────────────────────

func TestSO3_CompletePickingTask_AdvancesSOToCompleted(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	tenantID := "00000000-0000-0000-0000-000000000301"
	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-SO3A")
	seedInventory(t, db, "SKU-SO3A", "LOC-SO3A", 100, 10)
	customerID := seedCustomer(t, db, tenantID)

	// Create SO in 'submitted' status (as if Submit already ran).
	soID := seedSalesOrder(t, db, tenantID, customerID, userID)
	seedSalesOrderItem(t, db, soID, "SKU-SO3A", 20)

	// Create a picking task linked to the SO.
	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-SO3A", ExpectedQuantity: 20, Allocations: []database.LocationAllocation{
			{Location: "LOC-SO3A", Quantity: 20},
		}},
	}
	ptID := seedPickingTaskLinkedToSO(t, db, userID, soID, "in_progress", items)

	// Build PickingTaskRepository with SORepository wired (the fix under test).
	soRepo := &SalesOrdersRepository{DB: db}
	pickingRepo := &PickingTaskRepository{
		DB:           db,
		SORepository: soRepo,
	}

	resp := pickingRepo.CompletePickingTask(context.Background(), ptID, userID)
	require.Nil(t, resp, "CompletePickingTask should succeed")

	// SO must have advanced to 'completed' (all 20 units picked).
	status := getSalesOrderStatus(t, db, soID)
	assert.Equal(t, "completed", status, "SO status must be 'completed' after full picking")

	// Picking task itself must be completed.
	var ptStatus string
	require.NoError(t, db.Raw("SELECT status FROM picking_tasks WHERE id = ?", ptID).Scan(&ptStatus).Error)
	assert.Equal(t, "completed", ptStatus)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3b — Partial picking → SO advances to 'partial'
// ─────────────────────────────────────────────────────────────────────────────

func TestSO3_CompletePickingTask_AdvancesSOToPartial(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	tenantID := "00000000-0000-0000-0000-000000000302"
	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-SO3B")
	seedInventory(t, db, "SKU-SO3B", "LOC-SO3B", 100, 0)
	customerID := seedCustomer(t, db, tenantID)

	// SO requests 30, but picking only delivers 15.
	soID := seedSalesOrder(t, db, tenantID, customerID, userID)
	seedSalesOrderItem(t, db, soID, "SKU-SO3B", 30)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-SO3B", ExpectedQuantity: 30, Allocations: []database.LocationAllocation{
			{Location: "LOC-SO3B", Quantity: 15},
		}},
	}
	ptID := seedPickingTaskLinkedToSO(t, db, userID, soID, "in_progress", items)

	soRepo := &SalesOrdersRepository{DB: db}
	pickingRepo := &PickingTaskRepository{
		DB:           db,
		SORepository: soRepo,
	}

	resp := pickingRepo.CompletePickingTask(context.Background(), ptID, userID)
	require.Nil(t, resp, "CompletePickingTask should succeed")

	// Only 15 of 30 were picked → partial.
	status := getSalesOrderStatus(t, db, soID)
	assert.Equal(t, "partial", status, "SO status must be 'partial' after incomplete picking")
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3c — nil SORepository (pre-fix regression guard) silently skips SO update
// ─────────────────────────────────────────────────────────────────────────────

func TestSO3_NilSORepository_DoesNotPanic(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	tenantID := "00000000-0000-0000-0000-000000000303"
	userID := seedUser(t, db)
	seedArticle(t, db, "SKU-SO3C")
	seedInventory(t, db, "SKU-SO3C", "LOC-SO3C", 50, 0)
	customerID := seedCustomer(t, db, tenantID)

	soID := seedSalesOrder(t, db, tenantID, customerID, userID)
	seedSalesOrderItem(t, db, soID, "SKU-SO3C", 10)

	items := []requests.PickingTaskItemRequest{
		{SKU: "SKU-SO3C", ExpectedQuantity: 10, Allocations: []database.LocationAllocation{
			{Location: "LOC-SO3C", Quantity: 10},
		}},
	}
	ptID := seedPickingTaskLinkedToSO(t, db, userID, soID, "in_progress", items)

	// SORepository deliberately nil — must not panic (nil guard in CompletePickingTask).
	pickingRepo := &PickingTaskRepository{
		DB:           db,
		SORepository: nil,
	}

	resp := pickingRepo.CompletePickingTask(context.Background(), ptID, userID)
	require.Nil(t, resp, "CompletePickingTask should succeed even without SORepository")

	// SO stays in 'submitted' — the nil guard skips the update silently.
	status := getSalesOrderStatus(t, db, soID)
	assert.Equal(t, "submitted", status, "SO must stay 'submitted' when SORepository is nil")
}
