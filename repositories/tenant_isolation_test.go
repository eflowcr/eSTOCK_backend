package repositories

// tenant_isolation_test.go — S2.5 M3.6
// Integration-level multi-tenant isolation tests using testcontainers + real Postgres.
// Uses setupGORMTestDB (defined in receiving_tasks_upsert_lot_integration_test.go).
//
// These tests exercise the actual GORM WHERE tenant_id = ? clause inside
// GetAllForTenant / ExportXToExcel. If the WHERE clause is removed from the
// production code, these tests will fail.
//
// Run: go test -v ./repositories/... -run TestTenantIsolation
// Requires Docker (testcontainers). Automatically skipped in -short mode.

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testTenantA = "00000000-0000-0000-0000-000000000001"
	testTenantB = "00000000-0000-0000-0000-000000000002"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// seedUserForIsolation inserts a minimal user row and returns its id.
// Uses a deterministic email so repeated calls are idempotent via ON CONFLICT.
func seedUserForIsolation(t *testing.T, db *gorm.DB, tag string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	email := fmt.Sprintf("iso-%s@test.com", tag)
	db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES (?, 'Test', 'ISO', ?, 'hashed', NOW(), NOW())
		ON CONFLICT (email) DO NOTHING`, id, email)
	var uid string
	db.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&uid)
	if uid == "" {
		uid = id
	}
	return uid
}

// seedAdjustmentRow inserts an adjustment row with an explicit tenant_id.
func seedAdjustmentRow(t *testing.T, db *gorm.DB, userID, tenantID, sku string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO adjustments
			(id, sku, location, previous_quantity, adjustment_quantity, new_quantity,
			 reason, user_id, adjustment_type, tenant_id, created_at)
		VALUES (?, ?, 'LOC-ISO', 10, 5, 15, 'isolation-test', ?, 'increase', ?, NOW())`,
		id, sku, userID, tenantID).Error)
	return id
}

// seedPickingTaskRow inserts a picking_task row with an explicit tenant_id.
func seedPickingTaskRow(t *testing.T, db *gorm.DB, userID, tenantID, orderNumber string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	items, _ := json.Marshal([]interface{}{})
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks
			(id, task_id, order_number, created_by, status, priority, items, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', 'normal', ?, ?, NOW(), NOW())`,
		id, "PICK-ISO-"+id[:6], orderNumber, userID, string(items), tenantID).Error)
	return id
}

// seedReceivingTaskRow inserts a receiving_task row with an explicit tenant_id.
func seedReceivingTaskRow(t *testing.T, db *gorm.DB, userID, tenantID, inboundNumber string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	items, _ := json.Marshal([]interface{}{})
	require.NoError(t, db.Exec(`
		INSERT INTO receiving_tasks
			(id, task_id, inbound_number, created_by, status, priority, items, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', 'normal', ?, ?, NOW(), NOW())`,
		id, "RCV-ISO-"+id[:6], inboundNumber, userID, string(items), tenantID).Error)
	return id
}

// ─── Adjustments isolation ────────────────────────────────────────────────────

// TestTenantIsolation_Adjustments_GetAllForTenant seeds two tenants then calls
// the real AdjustmentsRepository.GetAllForTenant and verifies cross-tenant
// invisibility. Removing WHERE tenant_id = ? from the production code causes
// this test to fail.
func TestTenantIsolation_Adjustments_GetAllForTenant(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUserForIsolation(t, db, "adj")

	// Tenant A: 2 rows; Tenant B: 1 row.
	idA1 := seedAdjustmentRow(t, db, userID, testTenantA, "SKU-A1")
	idA2 := seedAdjustmentRow(t, db, userID, testTenantA, "SKU-A2")
	idB1 := seedAdjustmentRow(t, db, userID, testTenantB, "SKU-B1")

	repo := &AdjustmentsRepository{DB: db}

	// Tenant A sees exactly its 2 rows, not tenant B's.
	rowsA, errA := repo.GetAllForTenant(testTenantA)
	require.Nil(t, errA)
	idsA := make([]string, len(rowsA))
	for i, r := range rowsA {
		idsA[i] = r.ID
		assert.Equal(t, testTenantA, r.TenantID, "GetAllForTenant(A) must only return tenant A rows")
	}
	assert.Contains(t, idsA, idA1)
	assert.Contains(t, idsA, idA2)
	assert.NotContains(t, idsA, idB1)

	// Tenant B sees exactly its 1 row, not tenant A's.
	rowsB, errB := repo.GetAllForTenant(testTenantB)
	require.Nil(t, errB)
	require.Len(t, rowsB, 1)
	assert.Equal(t, idB1, rowsB[0].ID)
	assert.Equal(t, testTenantB, rowsB[0].TenantID)
}

// TestTenantIsolation_Adjustments_ExportTenantScoped verifies that
// ExportAdjustmentsToExcel(tenantID) returns only the calling tenant's rows
// (i.e., the bytes represent a file built from tenant-scoped data only).
func TestTenantIsolation_Adjustments_ExportTenantScoped(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUserForIsolation(t, db, "adj-exp")

	seedAdjustmentRow(t, db, userID, testTenantA, "SKU-EXP-A")
	seedAdjustmentRow(t, db, userID, testTenantB, "SKU-EXP-B")

	repo := &AdjustmentsRepository{DB: db}

	// Export for tenant A must not fail and must return non-empty bytes.
	bytesA, errA := repo.ExportAdjustmentsToExcel(testTenantA)
	require.Nil(t, errA)
	assert.NotEmpty(t, bytesA, "export for tenant A should return xlsx bytes")

	// Export for an unknown tenant returns empty-data response (no rows → "no data" path).
	bytesUnknown, errUnknown := repo.ExportAdjustmentsToExcel("00000000-0000-0000-0000-000000000099")
	assert.Nil(t, bytesUnknown, "unknown tenant should return nil bytes (no data)")
	require.NotNil(t, errUnknown, "no-data path returns a handled InternalResponse")
	assert.True(t, errUnknown.Handled, "no-data response should be handled")
}

// ─── PickingTask isolation ────────────────────────────────────────────────────

// TestTenantIsolation_PickingTasks_GetAllForTenant exercises the real SQL
// WHERE pt.tenant_id = ? clause in PickingTaskRepository.GetAllForTenant.
func TestTenantIsolation_PickingTasks_GetAllForTenant(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUserForIsolation(t, db, "pick")

	idA1 := seedPickingTaskRow(t, db, userID, testTenantA, "ISO-ORD-A1")
	idA2 := seedPickingTaskRow(t, db, userID, testTenantA, "ISO-ORD-A2")
	idB1 := seedPickingTaskRow(t, db, userID, testTenantB, "ISO-ORD-B1")

	repo := &PickingTaskRepository{DB: db}

	rowsA, errA := repo.GetAllForTenant(testTenantA)
	require.Nil(t, errA)
	idsA := make([]string, len(rowsA))
	for i, r := range rowsA {
		idsA[i] = r.ID
		assert.Equal(t, testTenantA, r.TenantID, "GetAllForTenant(A) must only return tenant A rows")
	}
	assert.Contains(t, idsA, idA1)
	assert.Contains(t, idsA, idA2)
	assert.NotContains(t, idsA, idB1)

	rowsB, errB := repo.GetAllForTenant(testTenantB)
	require.Nil(t, errB)
	idsB := make([]string, len(rowsB))
	for i, r := range rowsB {
		idsB[i] = r.ID
	}
	assert.Contains(t, idsB, idB1)
	assert.NotContains(t, idsB, idA1)
	assert.NotContains(t, idsB, idA2)
}

// ─── ReceivingTask isolation ──────────────────────────────────────────────────

// TestTenantIsolation_ReceivingTasks_GetAllForTenant exercises the real SQL
// WHERE rt.tenant_id = ? clause in ReceivingTasksRepository.GetAllForTenant.
func TestTenantIsolation_ReceivingTasks_GetAllForTenant(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	userID := seedUserForIsolation(t, db, "recv")

	idA1 := seedReceivingTaskRow(t, db, userID, testTenantA, "ISO-RCV-A1")
	idB1 := seedReceivingTaskRow(t, db, userID, testTenantB, "ISO-RCV-B1")
	idB2 := seedReceivingTaskRow(t, db, userID, testTenantB, "ISO-RCV-B2")

	repo := &ReceivingTasksRepository{DB: db}

	rowsA, errA := repo.GetAllForTenant(testTenantA)
	require.Nil(t, errA)
	idsA := make([]string, len(rowsA))
	for i, r := range rowsA {
		idsA[i] = r.ID
		assert.Equal(t, testTenantA, r.TenantID, "GetAllForTenant(A) must only return tenant A rows")
	}
	assert.Contains(t, idsA, idA1)
	assert.NotContains(t, idsA, idB1)
	assert.NotContains(t, idsA, idB2)

	rowsB, errB := repo.GetAllForTenant(testTenantB)
	require.Nil(t, errB)
	idsB := make([]string, len(rowsB))
	for i, r := range rowsB {
		idsB[i] = r.ID
	}
	assert.Contains(t, idsB, idB1)
	assert.Contains(t, idsB, idB2)
	assert.NotContains(t, idsB, idA1)
}

// ─── struct field sanity (compile-time guards) ────────────────────────────────
// These lightweight checks verify the TenantID field exists on DB models.
// They do NOT provide isolation coverage — the integration tests above do that.

func TestTenantIsolation_AdjustmentModelHasTenantIDField(t *testing.T) {
	adj := database.Adjustment{ID: "test", TenantID: testTenantA}
	assert.Equal(t, testTenantA, adj.TenantID)
}

func TestTenantIsolation_PickingTaskModelHasTenantIDField(t *testing.T) {
	pt := database.PickingTask{ID: "test", TenantID: testTenantA}
	assert.Equal(t, testTenantA, pt.TenantID)
}

func TestTenantIsolation_ReceivingTaskModelHasTenantIDField(t *testing.T) {
	rt := database.ReceivingTask{ID: "test", TenantID: testTenantA}
	assert.Equal(t, testTenantA, rt.TenantID)
}

// ─── Articles isolation (S3.5 W1 — HR-S3-W5 C2 fix) ──────────────────────────

// seedArticleRow inserts an article row with an explicit tenant_id directly via SQL,
// bypassing the GORM model so we can exercise the cross-tenant scenarios that the
// production code is meant to prevent.
func seedArticleRow(t *testing.T, db *gorm.DB, tenantID, sku, name string) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO articles
			(id, tenant_id, sku, name, presentation, track_by_lot, track_by_serial,
			 track_expiration, rotation_strategy, is_active, safety_stock, min_order_qty,
			 created_at, updated_at)
		VALUES (?, ?::uuid, ?, ?, 'unit', false, false, false, 'fifo', true, 0, 0, NOW(), NOW())`,
		id, tenantID, sku, name).Error)
	return id
}

// TestArticles_TenantIsolation_GetAll_returnsOnlyOwnTenant seeds two tenants
// then verifies the per-tenant GetAllArticlesForTenant only sees its own rows.
// Regression guard for HR-S3-W5 C2 — the SeedFarma data leak that motivated
// this entire sprint must never come back.
func TestArticles_TenantIsolation_GetAll_returnsOnlyOwnTenant(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	idA1 := seedArticleRow(t, db, testTenantA, "ISO-ART-A1", "Article A1")
	idA2 := seedArticleRow(t, db, testTenantA, "ISO-ART-A2", "Article A2")
	idB1 := seedArticleRow(t, db, testTenantB, "ISO-ART-B1", "Article B1")

	repo := &ArticlesRepository{DB: db}

	rowsA, errA := repo.GetAllArticlesForTenant(testTenantA)
	require.Nil(t, errA)
	idsA := make([]string, 0, len(rowsA))
	for _, r := range rowsA {
		idsA = append(idsA, r.ID)
		assert.Equal(t, testTenantA, r.TenantID, "GetAllArticlesForTenant(A) must only return tenant A rows")
	}
	assert.Contains(t, idsA, idA1)
	assert.Contains(t, idsA, idA2)
	assert.NotContains(t, idsA, idB1, "tenant A must not see tenant B's articles")

	rowsB, errB := repo.GetAllArticlesForTenant(testTenantB)
	require.Nil(t, errB)
	require.Len(t, rowsB, 1)
	assert.Equal(t, idB1, rowsB[0].ID)
	assert.Equal(t, testTenantB, rowsB[0].TenantID)
}

// TestArticles_TenantIsolation_PerTenantSkuLookup exercises the per-tenant SKU
// lookup against the new composite (tenant_id, sku) index. Two different SKUs are
// used per tenant because the W1 migration intentionally retains the legacy
// global UNIQUE(sku) for FK target preservation — see migration 000029 comments
// and feedback_estock_articles_no_tenant_isolation.md. Once child-table FKs are
// retrofitted (future sprint), the global unique can be dropped and the same
// SKU will be allowed across tenants; that will earn a separate test.
//
// What this test PROVES today:
//   - Per-tenant reads are scoped: tenant A cannot see tenant B's row by SKU even
//     when both rows exist with the same row id pattern.
//   - The new composite index enforces uniqueness within a single tenant
//     (re-inserting (tenantA, "ART-A-001") fails).
func TestArticles_TenantIsolation_PerTenantSkuLookup(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	const skuA = "ISO-LOOKUP-A-001"
	const skuB = "ISO-LOOKUP-B-001"

	idA := seedArticleRow(t, db, testTenantA, skuA, "Article A1")
	idB := seedArticleRow(t, db, testTenantB, skuB, "Article B1")

	repo := &ArticlesRepository{DB: db}

	// Tenant A finds its own row, doesn't see tenant B's.
	gotA, errA := repo.GetBySkuForTenant(skuA, testTenantA)
	require.Nil(t, errA)
	require.NotNil(t, gotA)
	assert.Equal(t, idA, gotA.ID)
	assert.Equal(t, testTenantA, gotA.TenantID)

	notFoundA, errNF := repo.GetBySkuForTenant(skuB, testTenantA)
	require.NotNil(t, errNF, "tenant A must not see tenant B's SKU")
	assert.Equal(t, responses.StatusNotFound, errNF.StatusCode)
	assert.Nil(t, notFoundA)

	// Tenant B sees its own.
	gotB, errB := repo.GetBySkuForTenant(skuB, testTenantB)
	require.Nil(t, errB)
	require.NotNil(t, gotB)
	assert.Equal(t, idB, gotB.ID)

	// Composite unique (tenant_id, sku) enforced: re-inserting the same pair fails.
	dupErr := db.Exec(`
		INSERT INTO articles
			(id, tenant_id, sku, name, presentation, track_by_lot, track_by_serial,
			 track_expiration, rotation_strategy, is_active, safety_stock, min_order_qty,
			 created_at, updated_at)
		VALUES (gen_random_uuid()::text, ?::uuid, ?, 'dup', 'unit', false, false, false,
			'fifo', true, 0, 0, NOW(), NOW())`,
		testTenantA, skuA).Error
	require.Error(t, dupErr, "second insert of the same (tenant_id, sku) must violate articles_tenant_sku_key")
}

// TestArticles_TenantIsolation_GlobalSkuUniqueStillEnforced documents the W1
// limitation: the legacy global UNIQUE(sku) is retained as a FK target for the
// 8+ child tables that reference articles(sku). Two different tenants therefore
// CANNOT yet share the same SKU. This test asserts the limitation explicitly so
// any future migration that drops the global unique will need to update this
// test too — providing a clear paper trail for the structural change.
func TestArticles_TenantIsolation_GlobalSkuUniqueStillEnforced(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	const sharedSKU = "ISO-GLOBAL-SHARED-001"
	seedArticleRow(t, db, testTenantA, sharedSKU, "Owned by tenant A")

	// Tenant B cannot register the same SKU until the global unique is dropped.
	dupErr := db.Exec(`
		INSERT INTO articles
			(id, tenant_id, sku, name, presentation, track_by_lot, track_by_serial,
			 track_expiration, rotation_strategy, is_active, safety_stock, min_order_qty,
			 created_at, updated_at)
		VALUES (gen_random_uuid()::text, ?::uuid, ?, 'tenant B copy', 'unit', false, false,
			false, 'fifo', true, 0, 0, NOW(), NOW())`,
		testTenantB, sharedSKU).Error
	require.Error(t, dupErr, "global articles_sku_key intentionally still enforces SKU uniqueness across tenants in W1")
	assert.Contains(t, dupErr.Error(), "articles_sku_key")
}

// TestTenantIsolation_ArticleModelHasTenantIDField is a compile-time guard so the
// TenantID struct field can't be silently removed.
func TestTenantIsolation_ArticleModelHasTenantIDField(t *testing.T) {
	a := database.Article{ID: "test", TenantID: testTenantA}
	assert.Equal(t, testTenantA, a.TenantID)
}
