// Integration tests for Purchase Orders (S3 W2-A: PO1+PO2+PO3).
// Requires Docker (testcontainers). Skipped automatically in -short mode.
// Run: go test -v ./repositories/... -run "TestPurchaseOrders"

package repositories

import (
	"encoding/json"
	"fmt"
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
// Integration test helpers
// ─────────────────────────────────────────────────────────────────────────────

// newPORepo returns a PurchaseOrdersRepository for the given test DB.
func newPORepo(db *gorm.DB) *PurchaseOrdersRepository {
	return &PurchaseOrdersRepository{DB: db}
}

// seedSupplier inserts a client of type 'supplier' and returns its id.
func seedSupplier(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO clients (id, tenant_id, type, code, name, is_active, created_at, updated_at)
		VALUES (?, ?, 'supplier', ?, ?, true, NOW(), NOW())`,
		id, tenantID, "SUP-"+id[:6], "Supplier "+id[:6]).Error)
	return id
}

// seedArticleForPO inserts a minimal article row if not already present.
func seedArticleForPO(t *testing.T, db *gorm.DB, sku string) {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	db.Exec(`
		INSERT INTO articles (id, sku, name, track_by_lot, track_by_serial, created_at, updated_at)
		VALUES (?, ?, ?, false, false, NOW(), NOW())
		ON CONFLICT (sku) DO NOTHING`, id, sku, "Article "+sku)
}

// seedUserForPO inserts a minimal user and returns their id.
func seedUserForPO(t *testing.T, db *gorm.DB, suffix string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	email := id + suffix + "@po-test.com"
	db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES (?, 'PO', 'User', ?, 'hashed', NOW(), NOW())
		ON CONFLICT (email) DO NOTHING`, id, email)
	var uid string
	db.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&uid)
	if uid == "" {
		uid = id
	}
	return uid
}

const integTenantID = "00000000-0000-0000-0000-000000000099"

// ─────────────────────────────────────────────────────────────────────────────
// PO1 — CRUD Integration Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrders_Create_Integration(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-create")
	seedArticleForPO(t, db, "SKU-PO-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items: []requests.CreatePurchaseOrderItemRequest{
			{ArticleSKU: "SKU-PO-001", ExpectedQty: 50},
		},
	}

	view, resp := repo.Create(integTenantID, userID, req)
	require.Nil(t, resp)
	require.NotNil(t, view)

	assert.NotEmpty(t, view.ID)
	assert.Equal(t, "draft", view.Status)
	assert.Equal(t, supplierID, view.SupplierID)
	assert.Len(t, view.Items, 1)
	assert.Equal(t, "SKU-PO-001", view.Items[0].ArticleSKU)
	assert.Equal(t, float64(50), view.Items[0].ExpectedQty)
	// PO number format: PO-YYYY-NNNN
	assert.Regexp(t, `^PO-\d{4}-\d{4}$`, view.PONumber)
}

func TestPurchaseOrders_PONumber_Sequenced_PerTenant(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	tenantA := "00000000-0000-0000-0000-000000000010"
	tenantB := "00000000-0000-0000-0000-000000000011"
	supplierA := seedSupplier(t, db, tenantA)
	supplierB := seedSupplier(t, db, tenantB)
	userID := seedUserForPO(t, db, "-seq")
	seedArticleForPO(t, db, "SKU-SEQ-001")

	repo := newPORepo(db)
	baseReq := func(supplierID string) *requests.CreatePurchaseOrderRequest {
		return &requests.CreatePurchaseOrderRequest{
			SupplierID: supplierID,
			Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-SEQ-001", ExpectedQty: 1}},
		}
	}

	// Tenant A: 2 POs
	v1, _ := repo.Create(tenantA, userID, baseReq(supplierA))
	v2, _ := repo.Create(tenantA, userID, baseReq(supplierA))
	// Tenant B: 1 PO — should start at 0001 independently
	v3, _ := repo.Create(tenantB, userID, baseReq(supplierB))

	year := time.Now().Year()
	assert.Equal(t, v1.PONumber, "PO-"+itoa(year)+"-0001")
	assert.Equal(t, v2.PONumber, "PO-"+itoa(year)+"-0002")
	assert.Equal(t, v3.PONumber, "PO-"+itoa(year)+"-0001") // independent sequence
}

func TestPurchaseOrders_GetByID_TenantIsolation(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	tenantA := "00000000-0000-0000-0000-000000000020"
	tenantB := "00000000-0000-0000-0000-000000000021"
	supplierA := seedSupplier(t, db, tenantA)
	userID := seedUserForPO(t, db, "-iso")
	seedArticleForPO(t, db, "SKU-ISO-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierA,
		Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-ISO-001", ExpectedQty: 5}},
	}

	created, _ := repo.Create(tenantA, userID, req)
	require.NotNil(t, created)

	// Tenant A can read its own PO
	found, resp := repo.GetByID(created.ID, tenantA)
	require.Nil(t, resp)
	assert.Equal(t, created.ID, found.ID)

	// Tenant B cannot read Tenant A's PO
	notFound, resp := repo.GetByID(created.ID, tenantB)
	assert.Nil(t, notFound)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

func TestPurchaseOrders_Update_DraftOnly(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-upd")
	seedArticleForPO(t, db, "SKU-UPD-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-UPD-001", ExpectedQty: 10}},
	}
	created, _ := repo.Create(integTenantID, userID, req)
	require.NotNil(t, created)

	notes := "updated notes"
	updated, resp := repo.Update(created.ID, integTenantID, &requests.UpdatePurchaseOrderRequest{Notes: &notes})
	require.Nil(t, resp)
	require.NotNil(t, updated)
	assert.Equal(t, "updated notes", *updated.Notes)
}

func TestPurchaseOrders_SoftDelete(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-del")
	seedArticleForPO(t, db, "SKU-DEL-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-DEL-001", ExpectedQty: 5}},
	}
	created, _ := repo.Create(integTenantID, userID, req)
	require.NotNil(t, created)

	resp := repo.SoftDelete(created.ID, integTenantID)
	require.Nil(t, resp)

	// After soft delete, GetByID should return not-found.
	found, resp := repo.GetByID(created.ID, integTenantID)
	assert.Nil(t, found)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

// ─────────────────────────────────────────────────────────────────────────────
// PO2 — Lifecycle: Submit → auto-creates receiving task
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrders_Submit_CreatesReceivingTask(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-sub")
	seedArticleForPO(t, db, "SKU-SUB-001")
	seedArticleForPO(t, db, "SKU-SUB-002")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items: []requests.CreatePurchaseOrderItemRequest{
			{ArticleSKU: "SKU-SUB-001", ExpectedQty: 20},
			{ArticleSKU: "SKU-SUB-002", ExpectedQty: 15},
		},
	}
	created, _ := repo.Create(integTenantID, userID, req)
	require.NotNil(t, created)

	// Submit
	poView, rtID, resp := repo.Submit(created.ID, integTenantID, userID)
	require.Nil(t, resp)
	require.NotNil(t, poView)
	require.NotEmpty(t, rtID)

	assert.Equal(t, "submitted", poView.Status)
	assert.NotNil(t, poView.SubmittedAt)
	assert.Equal(t, rtID, *poView.ReceivingTaskID)

	// Verify receiving task was created in DB.
	var rtCount int64
	db.Raw("SELECT COUNT(*) FROM receiving_tasks WHERE id = ? AND tenant_id = ?", rtID, integTenantID).Scan(&rtCount)
	assert.Equal(t, int64(1), rtCount, "receiving task should exist in DB")

	// Verify receiving_task has purchase_order_id link.
	var poid string
	db.Raw("SELECT purchase_order_id FROM receiving_tasks WHERE id = ?", rtID).Scan(&poid)
	assert.Equal(t, created.ID, poid)

	// Verify RT items contain both SKUs.
	var itemsJSON []byte
	db.Raw("SELECT items FROM receiving_tasks WHERE id = ?", rtID).Scan(&itemsJSON)
	var items []map[string]interface{}
	require.NoError(t, json.Unmarshal(itemsJSON, &items))
	assert.Len(t, items, 2)
}

func TestPurchaseOrders_Submit_RejectsDraftAlreadySubmitted(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-sub2")
	seedArticleForPO(t, db, "SKU-SUB3-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-SUB3-001", ExpectedQty: 5}},
	}
	created, _ := repo.Create(integTenantID, userID, req)
	// First submit — should succeed.
	_, _, resp := repo.Submit(created.ID, integTenantID, userID)
	require.Nil(t, resp)

	// Second submit — should fail (not draft).
	_, _, resp = repo.Submit(created.ID, integTenantID, userID)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

func TestPurchaseOrders_Cancel_Transitions(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-can")
	seedArticleForPO(t, db, "SKU-CAN-001")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items:      []requests.CreatePurchaseOrderItemRequest{{ArticleSKU: "SKU-CAN-001", ExpectedQty: 10}},
	}
	created, _ := repo.Create(integTenantID, userID, req)

	// Cancel a draft PO.
	cancelled, resp := repo.Cancel(created.ID, integTenantID)
	require.Nil(t, resp)
	assert.Equal(t, "cancelled", cancelled.Status)
	assert.NotNil(t, cancelled.CancelledAt)

	// Cannot cancel again.
	_, resp = repo.Cancel(created.ID, integTenantID)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
}

// ─────────────────────────────────────────────────────────────────────────────
// PO3 — Receiving auto-link: UpdateReceivedQty advances PO status
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrders_UpdateReceivedQty_PartialThenComplete(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	supplierID := seedSupplier(t, db, integTenantID)
	userID := seedUserForPO(t, db, "-qty")
	seedArticleForPO(t, db, "SKU-QTY-001")
	seedArticleForPO(t, db, "SKU-QTY-002")

	repo := newPORepo(db)
	req := &requests.CreatePurchaseOrderRequest{
		SupplierID: supplierID,
		Items: []requests.CreatePurchaseOrderItemRequest{
			{ArticleSKU: "SKU-QTY-001", ExpectedQty: 10},
			{ArticleSKU: "SKU-QTY-002", ExpectedQty: 5},
		},
	}
	created, _ := repo.Create(integTenantID, userID, req)
	require.NotNil(t, created)

	// Submit first.
	_, _, _ = repo.Submit(created.ID, integTenantID, userID)

	// Partially fulfill SKU-QTY-001 only (5 of 10).
	resp := repo.UpdateReceivedQty(created.ID, []database.PurchaseOrderItemQtyUpdate{
		{ArticleSKU: "SKU-QTY-001", ReceivedQty: 5, RejectedQty: 0},
	})
	require.Nil(t, resp)

	// Status should be 'partial' now.
	view, _ := repo.GetByID(created.ID, integTenantID)
	assert.Equal(t, "partial", view.Status)

	// Fully fulfill remaining.
	resp = repo.UpdateReceivedQty(created.ID, []database.PurchaseOrderItemQtyUpdate{
		{ArticleSKU: "SKU-QTY-001", ReceivedQty: 5, RejectedQty: 0},
		{ArticleSKU: "SKU-QTY-002", ReceivedQty: 0, RejectedQty: 5}, // all rejected counts as fulfilled
	})
	require.Nil(t, resp)

	view, _ = repo.GetByID(created.ID, integTenantID)
	assert.Equal(t, "completed", view.Status)
	assert.NotNil(t, view.CompletedAt)
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
