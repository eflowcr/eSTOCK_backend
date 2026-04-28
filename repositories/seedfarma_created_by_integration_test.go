// Integration tests for S3.5.3 N3 — SeedFarma created_by + repository defensive
// JOIN. Two scenarios are covered, each in a separate Postgres testcontainer:
//
//  1. SeedFarma stamps the supplied adminUserID on receiving + picking
//     created_by, NOT the tenantID (Fix A regression guard).
//  2. GetAllForTenant on receiving + picking still returns rows whose
//     created_by points at a non-existent users.id (Fix B regression guard:
//     the LEFT JOIN must not silently drop the row).
//
// Both tests skip in -short mode (Docker required) via setupGORMTestDB.

package repositories

import (
	"context"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// insertTenant inserts a tenants row so demo_data_seeds (FK on tenants) and
// the multi-tenant queries see a valid parent. Idempotent on (id).
func insertTenant(t *testing.T, db *gorm.DB, tenantID, slug, email string) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO tenants (id, name, slug, email, status, trial_ends_at, is_active)
		VALUES (?::uuid, ?, ?, ?, 'trial', NOW() + interval '14 days', true)
		ON CONFLICT (id) DO NOTHING`,
		tenantID, slug, slug, email).Error)
}

// insertUserForTenant inserts a real users row owned by tenantID and returns
// its id. Distinct from the package-shared seedUser helper (which doesn't
// stamp tenant_id) to keep this test self-contained against multi-tenant
// schema changes. Reuses the admin role seeded by seedAdminRole (signup tests).
func insertUserForTenant(t *testing.T, db *gorm.DB, tenantID, email, first, last string) string {
	t.Helper()
	roleID := seedAdminRole(t, db) // idempotent
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, password, role_id, is_active, created_at, updated_at)
		VALUES (?, ?::uuid, ?, ?, ?, 'hashed', ?, true, NOW(), NOW())
		ON CONFLICT (email) DO NOTHING`,
		id, tenantID, first, last, email, roleID).Error)
	var uid string
	require.NoError(t, db.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&uid).Error)
	require.NotEmpty(t, uid, "insertUserForTenant produced no user row")
	return uid
}

// ─── Test 1 — Fix A regression guard ──────────────────────────────────────────

// TestSeedFarma_AssignsAdminUserIDAsCreatedBy proves that when a real adminUserID
// is threaded through SeedFarma, every receiving + picking demo row carries that
// id in created_by — NOT the tenantID. This is the root cause of N3: the
// previous implementation passed tenantID, the repository INNER-JOINed users on
// created_by, and the freshly-signed-up tenant's dashboard came up empty.
func TestSeedFarma_AssignsAdminUserIDAsCreatedBy(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()

	const tenantID = "33333333-3333-3333-3333-333333333333"
	insertTenant(t, db, tenantID, "createdby-tenant", "createdby@test.com")

	// A real admin user owned by this tenant.
	adminID := insertUserForTenant(t, db, tenantID, "admin-createdby@test.com", "Created", "Admin")

	require.NoError(t, tools.SeedFarma(ctx, db, tenantID, adminID),
		"SeedFarma must succeed when given a real adminUserID")

	// Every demo receiving_task for this tenant must reference adminID, not tenantID.
	var rtRowsWithAdmin int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM receiving_tasks WHERE tenant_id = ? AND created_by = ?`,
		tenantID, adminID,
	).Scan(&rtRowsWithAdmin).Error)
	assert.EqualValues(t, 20, rtRowsWithAdmin,
		"all 20 demo receiving_tasks must carry created_by = adminUserID")

	var rtRowsWithTenant int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM receiving_tasks WHERE tenant_id = ? AND created_by = ?`,
		tenantID, tenantID,
	).Scan(&rtRowsWithTenant).Error)
	assert.EqualValues(t, 0, rtRowsWithTenant,
		"no demo receiving_task may carry created_by = tenantID (the N3 bug shape)")

	// Same for picking_tasks (15 demo rows).
	var ptRowsWithAdmin int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM picking_tasks WHERE tenant_id = ? AND created_by = ?`,
		tenantID, adminID,
	).Scan(&ptRowsWithAdmin).Error)
	assert.EqualValues(t, 15, ptRowsWithAdmin,
		"all 15 demo picking_tasks must carry created_by = adminUserID")

	var ptRowsWithTenant int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM picking_tasks WHERE tenant_id = ? AND created_by = ?`,
		tenantID, tenantID,
	).Scan(&ptRowsWithTenant).Error)
	assert.EqualValues(t, 0, ptRowsWithTenant,
		"no demo picking_task may carry created_by = tenantID")

	// End-to-end visibility: GetAllForTenant must now return the rows. This
	// closes the loop on N3 — the real symptom (empty dashboard) is exactly
	// "GetAllForTenant returned 0 rows" and it must NOT happen anymore.
	rtRepo := &ReceivingTasksRepository{DB: db}
	rts, errRT := rtRepo.GetAllForTenant(tenantID)
	require.Nil(t, errRT)
	assert.Len(t, rts, 20, "dashboard must surface all 20 demo receiving tasks (N3 symptom guard)")

	ptRepo := &PickingTaskRepository{DB: db}
	pts, errPT := ptRepo.GetAllForTenant(tenantID)
	require.Nil(t, errPT)
	assert.Len(t, pts, 15, "dashboard must surface all 15 demo picking tasks (N3 symptom guard)")
}

// ─── Test 2 — Fix B regression guard (receiving) ─────────────────────────────

// TestReceivingTasksRepo_GetAllForTenant_ReturnsRowsEvenWithMissingUser proves
// that the LEFT JOIN to users keeps a row visible when created_by points at a
// user id that no longer exists (e.g. user deleted, system_seed sentinel,
// pre-fix legacy data). The previous INNER JOIN dropped such rows silently.
func TestReceivingTasksRepo_GetAllForTenant_ReturnsRowsEvenWithMissingUser(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	const (
		tenantID    = "44444444-4444-4444-4444-444444444444"
		ghostUserID = "00000000-0000-0000-0000-deadbeef0001" // intentionally not in users
	)
	insertTenant(t, db, tenantID, "ghost-rt-tenant", "ghost-rt@test.com")

	// Insert a receiving_task whose created_by references no real user.
	taskUUID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO receiving_tasks
			(id, task_id, inbound_number, created_by, status, priority, items, tenant_id, created_at, updated_at)
		VALUES
			(?, ?, ?, ?, 'open', 'normal', '[]'::jsonb, ?::uuid, NOW(), NOW())`,
		taskUUID, "RT-GHOST-"+taskUUID[:4], "IN-GHOST-"+taskUUID[:4], ghostUserID, tenantID,
	).Error)

	repo := &ReceivingTasksRepository{DB: db}
	rows, errResp := repo.GetAllForTenant(tenantID)
	require.Nil(t, errResp)
	require.Len(t, rows, 1,
		"row with missing-user created_by must still surface — INNER JOIN regression guard")
	assert.Equal(t, ghostUserID, rows[0].CreatedBy,
		"created_by must round-trip even when the user is missing")
	assert.Empty(t, rows[0].UserCreatorName,
		"creator name must be empty (COALESCE on NULL JOIN side) — not error, not omit row")
}

// ─── Test 3 — Fix B regression guard (picking) ───────────────────────────────

// TestPickingTasksRepo_GetAllForTenant_ReturnsRowsEvenWithMissingUser is the
// picking-side mirror of the receiving test above. Same root cause, same fix,
// independently guarded so a future refactor of one repo can't silently
// regress the other.
func TestPickingTasksRepo_GetAllForTenant_ReturnsRowsEvenWithMissingUser(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	const (
		tenantID    = "55555555-5555-5555-5555-555555555555"
		ghostUserID = "00000000-0000-0000-0000-deadbeef0002"
	)
	insertTenant(t, db, tenantID, "ghost-pt-tenant", "ghost-pt@test.com")

	taskUUID, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks
			(id, task_id, order_number, created_by, status, priority, items, tenant_id, created_at, updated_at)
		VALUES
			(?, ?, ?, ?, 'open', 'normal', '[]'::jsonb, ?::uuid, NOW(), NOW())`,
		taskUUID, "PK-GHOST-"+taskUUID[:4], "ORD-GHOST-"+taskUUID[:4], ghostUserID, tenantID,
	).Error)

	repo := &PickingTaskRepository{DB: db}
	rows, errResp := repo.GetAllForTenant(tenantID)
	require.Nil(t, errResp)
	require.Len(t, rows, 1,
		"row with missing-user created_by must still surface — INNER JOIN regression guard")
	assert.Equal(t, ghostUserID, rows[0].CreatedBy)
	assert.Empty(t, rows[0].UserCreatorName)
}
