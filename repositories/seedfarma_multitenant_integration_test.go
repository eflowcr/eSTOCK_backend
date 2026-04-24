// Multi-tenant integration test for tools.SeedFarma — S3.5 W4.
//
// Requires Docker (testcontainers). Skipped automatically in `-short` mode by
// setupGORMTestDB. The unit-level helper coverage (tenantPrefix / prefixedSKU
// / prefixedTaskID) lives in tools/demo_seeder_farma_multitenant_test.go and
// runs without Docker.
//
// What this test PROVES end-to-end against real Postgres + every migration:
//
//  1. SeedFarma can be invoked for two distinct tenants in the same DB
//     without hitting articles_sku_key, receiving_tasks_task_id_key or
//     picking_tasks_task_id_key UNIQUE violations (the symptom that
//     motivated the W4 fix — see feedback_estock_articles_no_tenant_isolation.md).
//  2. Each tenant ends up with the same N-card demo dataset, but those
//     datasets are isolated rows (no shared article ids).
//  3. Cross-tenant SKU lookup returns NotFound (regression guard for the
//     SeedFarma data-leak that the prefixing scheme is designed to prevent).
//  4. demo_data_seeds carries one row per tenant (idempotency key is
//     (tenant_id, seed_name), set by migration 000023).

package repositories

import (
	"context"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedTenantRow inserts a row in tenants(id, …) so demo_data_seeds.tenant_id
// (FK → tenants.id) is satisfied. The default tenant from migration 000023 is
// already present; this is for the second tenant only.
func seedTenantRow(t *testing.T, db *gorm.DB, tenantID, slug, email string) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO tenants (id, name, slug, email, status, trial_ends_at, is_active)
		VALUES (?::uuid, ?, ?, ?, 'trial', NOW() + interval '14 days', true)
		ON CONFLICT (id) DO NOTHING`,
		tenantID, slug, slug, email).Error)
}

// TestSeedFarma_MultiTenantIsolation is the end-to-end W4 regression guard.
func TestSeedFarma_MultiTenantIsolation(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()

	const (
		tenantA = "00000000-0000-0000-0000-000000000001" // default tenant from 000023
		tenantB = "22222222-2222-2222-2222-222222222222"
	)

	// Tenant A is already present in the tenants table (default seed); only B
	// needs a row before SeedFarma touches FK-bound demo_data_seeds.
	seedTenantRow(t, db, tenantB, "tenantb-seedtest", "b@seedtest.com")

	// ── Seed both tenants ────────────────────────────────────────────────────
	require.NoError(t, tools.SeedFarma(ctx, db, tenantA), "seed for tenant A")
	require.NoError(t, tools.SeedFarma(ctx, db, tenantB), "seed for tenant B (must NOT collide on global UNIQUE indexes)")

	// ── Per-tenant article counts ────────────────────────────────────────────
	repo := &ArticlesRepository{DB: db}

	rowsA, errA := repo.GetAllArticlesForTenant(tenantA)
	require.Nil(t, errA)
	require.Len(t, rowsA, 50, "tenant A should see exactly its own 50 demo articles")

	rowsB, errB := repo.GetAllArticlesForTenant(tenantB)
	require.Nil(t, errB)
	require.Len(t, rowsB, 50, "tenant B should see exactly its own 50 demo articles (separate dataset)")

	// Every row returned by GetAllArticlesForTenant(X) must have TenantID == X.
	for _, r := range rowsA {
		assert.Equal(t, tenantA, r.TenantID, "tenant A leaks tenant B row %s", r.ID)
	}
	for _, r := range rowsB {
		assert.Equal(t, tenantB, r.TenantID, "tenant B leaks tenant A row %s", r.ID)
	}

	// Article IDs are disjoint — proves no FirstOrCreate-induced row sharing.
	idSetA := make(map[string]struct{}, len(rowsA))
	for _, r := range rowsA {
		idSetA[r.ID] = struct{}{}
	}
	for _, r := range rowsB {
		if _, leaked := idSetA[r.ID]; leaked {
			t.Fatalf("tenant B inherited tenant A's article id %s — multi-tenant data leak", r.ID)
		}
	}

	// ── Cross-tenant SKU lookup must return NotFound ─────────────────────────
	// Pick one SKU from tenant A's catalog and try to fetch it from tenant B.
	someSkuA := rowsA[0].SKU
	gotCross, errCross := repo.GetBySkuForTenant(someSkuA, tenantB)
	require.NotNil(t, errCross, "tenant B must not see tenant A's prefixed SKU %s", someSkuA)
	assert.Equal(t, responses.StatusNotFound, errCross.StatusCode)
	assert.Nil(t, gotCross)

	// Inverse direction.
	someSkuB := rowsB[0].SKU
	gotCross2, errCross2 := repo.GetBySkuForTenant(someSkuB, tenantA)
	require.NotNil(t, errCross2, "tenant A must not see tenant B's prefixed SKU %s", someSkuB)
	assert.Equal(t, responses.StatusNotFound, errCross2.StatusCode)
	assert.Nil(t, gotCross2)

	// ── demo_data_seeds tracks both tenants ──────────────────────────────────
	var seedCount int64
	require.NoError(t, db.Model(&database.DemoDataSeed{}).
		Where("seed_name = ?", tools.FarmaSeedName).Count(&seedCount).Error)
	assert.EqualValues(t, 2, seedCount, "demo_data_seeds should carry exactly one row per tenant")

	// ── Re-running SeedFarma for either tenant is a no-op (idempotency) ──────
	require.NoError(t, tools.SeedFarma(ctx, db, tenantA), "second seed call for tenant A must be idempotent")
	require.NoError(t, tools.SeedFarma(ctx, db, tenantB), "second seed call for tenant B must be idempotent")

	// Counts unchanged after re-run.
	var countA, countB int64
	require.NoError(t, db.Model(&database.Article{}).Where("tenant_id = ?", tenantA).Count(&countA).Error)
	require.NoError(t, db.Model(&database.Article{}).Where("tenant_id = ?", tenantB).Count(&countB).Error)
	assert.EqualValues(t, 50, countA, "re-run of SeedFarma for tenant A must NOT duplicate articles")
	assert.EqualValues(t, 50, countB, "re-run of SeedFarma for tenant B must NOT duplicate articles")
}
