// Integration tests for S3-W3-A: Delivery Notes + Backorders.
// Tests the full SO→picking→partial→backorder→fulfill flow.
//
// Requires Docker (testcontainers). Run: go test -v ./repositories/... -run TestDNBO
//
// All tests use setupGORMTestDB (defined in receiving_tasks_upsert_lot_integration_test.go).
package repositories

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// helpers (local to this file — complements helpers in other files)
// ─────────────────────────────────────────────────────────────────────────────

// dnTenantID is a fixed tenant UUID used across DN/BO integration tests.
const dnTenantID = "00000000-0000-0000-0000-000000000099"

func seedInventoryForDN(t *testing.T, db *gorm.DB, sku, location string, qty float64) {
	t.Helper()
	invID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO inventory (id, sku, location, quantity, reserved_qty, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, NOW(), NOW())
		ON CONFLICT (sku, location) DO UPDATE
		SET quantity = EXCLUDED.quantity, reserved_qty = 0, updated_at = NOW()`,
		invID, sku, location, qty).Error)
}

// seedSOForDN inserts a sales order with tenant_id set.
func seedSOForDN(t *testing.T, db *gorm.DB, custID, userID string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO sales_orders
			(id, tenant_id, so_number, customer_id, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'submitted', ?, NOW(), NOW())`,
		id, dnTenantID, "SO-DN-"+id[:6], custID, userID).Error)
	return id
}

// seedPickingTaskWithTenant inserts an in_progress picking_task with tenant + SO link.
func seedPickingTaskWithTenant(t *testing.T, db *gorm.DB, userID, soID string, items interface{}) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	itemsJSON, _ := json.Marshal(items)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks
			(id, task_id, order_number, created_by, status, priority, items,
			 sales_order_id, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'in_progress', 'normal', ?, ?, ?, NOW(), NOW())`,
		id, "T-"+id[:6], "ORD-"+id[:4], userID,
		string(itemsJSON), soID, dnTenantID).Error)
	return id
}

func getSOStatus(t *testing.T, db *gorm.DB, soID string) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw("SELECT status FROM sales_orders WHERE id = ?", soID).Scan(&status).Error)
	return status
}

func getDNsForSO(t *testing.T, db *gorm.DB, soID string) []database.DeliveryNote {
	t.Helper()
	var dns []database.DeliveryNote
	require.NoError(t, db.Where("sales_order_id = ?", soID).Find(&dns).Error)
	return dns
}

func getBOsForSO(t *testing.T, db *gorm.DB, soID string) []database.Backorder {
	t.Helper()
	var bos []database.Backorder
	require.NoError(t, db.Where("original_sales_order_id = ?", soID).Find(&bos).Error)
	return bos
}

// ─────────────────────────────────────────────────────────────────────────────
// TestDNBO1: picking completes fully → DN created, no backorders
// ─────────────────────────────────────────────────────────────────────────────

func TestDNBO1_FullPickingCreatesDN(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := seedUser(t, db)
	custID := seedCustomer(t, db, dnTenantID)
	sku := "SKU-DN01"

	seedArticle(t, db, sku)
	seedInventoryForDN(t, db, sku, "LOC-A", 20)

	soID := seedSOForDN(t, db, custID, userID)
	seedSalesOrderItem(t, db, soID, sku, 10)

	pickItems := []map[string]interface{}{
		{
			"sku":         sku,
			"required_qty": 10,
			"status":      "open",
			"allocations": []map[string]interface{}{
				{"location": "LOC-A", "quantity": 10, "picked_qty": 10},
			},
		},
	}
	pickID := seedPickingTaskWithTenant(t, db, userID, soID, pickItems)

	soRepo := &SalesOrdersRepository{DB: db}
	pickRepo := &PickingTaskRepository{DB: db, SORepository: soRepo}

	resp := pickRepo.CompletePickingTask(ctx, pickID, userID)
	require.Nil(t, resp, "CompletePickingTask should succeed")

	// Allow standalone DN transaction to complete.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "completed", getSOStatus(t, db, soID))

	dns := getDNsForSO(t, db, soID)
	assert.Len(t, dns, 1, "exactly 1 delivery note should be created")
	if len(dns) > 0 {
		assert.Equal(t, soID, dns[0].SalesOrderID)
		assert.Equal(t, pickID, *dns[0].PickingTaskID)
	}

	bos := getBOsForSO(t, db, soID)
	assert.Empty(t, bos, "no backorders for fully fulfilled SO")
}

// ─────────────────────────────────────────────────────────────────────────────
// TestDNBO2: partial picking → DN + backorder auto-generated
// ─────────────────────────────────────────────────────────────────────────────

func TestDNBO2_PartialPickingCreatesDNAndBackorder(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := seedUser(t, db)
	custID := seedCustomer(t, db, dnTenantID)
	sku := "SKU-DN02"

	seedArticle(t, db, sku)
	// Only 6 units in stock; SO expects 10.
	seedInventoryForDN(t, db, sku, "LOC-B", 6)

	soID := seedSOForDN(t, db, custID, userID)
	seedSalesOrderItem(t, db, soID, sku, 10)

	// Picking picks only 6 (partial).
	pickItems := []map[string]interface{}{
		{
			"sku":         sku,
			"required_qty": 10,
			"status":      "open",
			"allocations": []map[string]interface{}{
				{"location": "LOC-B", "quantity": 6, "picked_qty": 6},
			},
		},
	}
	pickID := seedPickingTaskWithTenant(t, db, userID, soID, pickItems)

	soRepo := &SalesOrdersRepository{DB: db}
	pickRepo := &PickingTaskRepository{DB: db, SORepository: soRepo}

	resp := pickRepo.CompletePickingTask(ctx, pickID, userID)
	require.Nil(t, resp)

	// Allow async transactions to settle.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "partial", getSOStatus(t, db, soID))

	dns := getDNsForSO(t, db, soID)
	assert.Len(t, dns, 1, "1 delivery note for the 6 picked units")

	bos := getBOsForSO(t, db, soID)
	require.Len(t, bos, 1, "1 backorder for remaining 4 units")
	assert.Equal(t, sku, bos[0].ArticleSKU)
	assert.Equal(t, "pending", bos[0].Status)
	assert.InDelta(t, 4.0, bos[0].RemainingQty, 0.01, "remaining_qty = 10 - 6 = 4")
}

// ─────────────────────────────────────────────────────────────────────────────
// TestDNBO3: BO2 Fulfill creates picking task with source_backorder_id set
// ─────────────────────────────────────────────────────────────────────────────

func TestDNBO3_FulfillBackorderCreatesPickingTask(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUser(t, db)
	custID := seedCustomer(t, db, dnTenantID)
	sku := "SKU-DN03"

	seedArticle(t, db, sku)
	// Stock available for backorder fulfillment.
	seedInventoryForDN(t, db, sku, "LOC-C", 5)

	soID := seedSOForDN(t, db, custID, userID)

	// Manually create a pending backorder.
	boID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO backorders
			(id, tenant_id, original_sales_order_id, article_sku, remaining_qty, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 5, 'pending', NOW(), NOW())`,
		boID, dnTenantID, soID, sku).Error)

	// InventoryRepository satisfies backorderInventorySuggestor directly.
	invRepo := &InventoryRepository{DB: db}
	boRepo := &BackordersRepository{DB: db, InventorySvc: invRepo}

	result, resp := boRepo.Fulfill(boID, dnTenantID, userID)
	require.Nil(t, resp, "Fulfill should succeed with stock available")
	require.NotNil(t, result)
	assert.NotEmpty(t, result.PickingTaskID)

	// Picking task must have source_backorder_id set (max-depth=1 guard).
	var pt database.PickingTask
	require.NoError(t, db.First(&pt, "id = ?", result.PickingTaskID).Error)
	require.NotNil(t, pt.SourceBackorderID, "source_backorder_id must be set")
	assert.Equal(t, boID, *pt.SourceBackorderID)

	// Backorder must have generated_picking_task_id updated.
	var bo database.Backorder
	require.NoError(t, db.First(&bo, "id = ?", boID).Error)
	require.NotNil(t, bo.GeneratedPickingTaskID)
	assert.Equal(t, result.PickingTaskID, *bo.GeneratedPickingTaskID)
}

// ─────────────────────────────────────────────────────────────────────────────
// TestDNBO4: backorder-sourced picking does NOT generate another backorder
// ─────────────────────────────────────────────────────────────────────────────

func TestDNBO4_BackorderSourcedPickingNoFurtherBackorder(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := seedUser(t, db)
	custID := seedCustomer(t, db, dnTenantID)
	sku := "SKU-DN04"

	seedArticle(t, db, sku)
	// Only 3 available (backorder was for 5, so partial again — but max depth=1).
	seedInventoryForDN(t, db, sku, "LOC-D", 3)

	soID := seedSOForDN(t, db, custID, userID)
	seedSalesOrderItem(t, db, soID, sku, 5)

	// Create the backorder.
	boID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO backorders
			(id, tenant_id, original_sales_order_id, article_sku, remaining_qty, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 5, 'pending', NOW(), NOW())`,
		boID, dnTenantID, soID, sku).Error)

	// Manually create a picking task sourced from this backorder (simulating BO2 fulfill).
	pickItems := []map[string]interface{}{
		{
			"sku":         sku,
			"required_qty": 3,
			"status":      "open",
			"allocations": []map[string]interface{}{
				{"location": "LOC-D", "quantity": 3, "picked_qty": 3},
			},
		},
	}
	pickID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	itemsJSON, _ := json.Marshal(pickItems)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks
			(id, task_id, order_number, created_by, status, priority, items,
			 sales_order_id, tenant_id, source_backorder_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'in_progress', 'high', ?, ?, ?, ?, NOW(), NOW())`,
		pickID, "T-BO-"+pickID[:4], "ORD-BO", userID,
		string(itemsJSON), soID, dnTenantID, boID).Error)

	soRepo := &SalesOrdersRepository{DB: db}
	pickRepo := &PickingTaskRepository{DB: db, SORepository: soRepo}

	resp := pickRepo.CompletePickingTask(ctx, pickID, userID)
	require.Nil(t, resp)

	time.Sleep(50 * time.Millisecond)

	// No NEW backorders should be created (max depth=1).
	bos := getBOsForSO(t, db, soID)
	// The pre-existing backorder (boID) is the only one.
	for _, bo := range bos {
		assert.Equal(t, boID, bo.ID, "only the original backorder should exist, no new ones")
	}

	// The original backorder should be updated with remaining qty reduced.
	var bo database.Backorder
	require.NoError(t, db.First(&bo, "id = ?", boID).Error)
	// picked 3 from remaining 5 → 2 remaining
	assert.InDelta(t, 2.0, bo.RemainingQty, 0.01, "remaining_qty should be reduced by 3")
	assert.Equal(t, "pending", bo.Status, "still pending with 2 remaining")
}
