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
