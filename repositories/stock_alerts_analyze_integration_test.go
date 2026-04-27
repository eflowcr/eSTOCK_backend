// stock_alerts_analyze_integration_test.go — S3.5.4 (B16 fix)
//
// Regression guard for the "Error al obtener el inventario" toast on /stock-alerts:
// Analyze() previously did `WHERE tenant_id = ?` directly on inventory and
// inventory_movements, but neither table has a tenant_id column today (deferred
// to S3.6 structural migration). The fix scopes via JOIN through articles.tenant_id.
//
// This test boots a real Postgres via testcontainers, runs the production
// migrations, seeds two tenants' worth of articles + inventory, and asserts:
//   - Analyze(tenantA) succeeds (no SQL error) and only sees tenant A inventory
//   - Tenant A's analyze run never inserts alerts referencing tenant B SKUs
//
// Run: go test -v ./repositories/... -run TestStockAlertsAnalyze
// Skipped under -short (no Docker required for unit gate).

package repositories

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedInventoryRow inserts an inventory row for a SKU. Note that inventory itself
// has no tenant_id column — multi-tenancy is enforced by the FK to articles.sku.
func seedInventoryRow(t *testing.T, db *gorm.DB, sku, location string, qty float64) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO inventory
			(id, sku, name, location, quantity, status, presentation, unit_price, created_at, updated_at)
		VALUES (?, ?, 'Test Item', ?, ?, 'available', 'unit', 1.50, NOW(), NOW())`,
		id, sku, location, qty).Error)
	return id
}

// TestStockAlertsAnalyze_TenantScopedViaArticlesJoin is the B16 regression guard.
// Before the fix, Analyze() failed with `column "tenant_id" does not exist` on
// inventory; this test crashes if anyone re-introduces that bug.
func TestStockAlertsAnalyze_TenantScopedViaArticlesJoin(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	// Seed articles for both tenants.
	seedArticleRow(t, db, testTenantA, "ANALYZE-SKU-A1", "Tenant A item")
	seedArticleRow(t, db, testTenantB, "ANALYZE-SKU-B1", "Tenant B item")

	// Seed inventory rows pointing at those SKUs. No tenant_id on this table —
	// scoping happens via the articles FK.
	seedInventoryRow(t, db, "ANALYZE-SKU-A1", "LOC-A", 5)
	seedInventoryRow(t, db, "ANALYZE-SKU-B1", "LOC-B", 7)

	repo := &StockAlertsRepository{DB: db}

	// Tenant A: must NOT error with the old "Error al obtener el inventario" message.
	respA, errA := repo.Analyze(testTenantA)
	require.Nil(t, errA, "Analyze must succeed without SQL error (B16 regression)")
	_ = respA // alerts may be empty for healthy stock; what matters is no error.

	// Verify any alerts written are scoped to tenant A only — no leak from B.
	var alertsA []database.StockAlert
	require.NoError(t, db.Table("stock_alerts").Where("tenant_id = ?", testTenantA).Find(&alertsA).Error)
	for _, a := range alertsA {
		assert.NotEqual(t, "ANALYZE-SKU-B1", a.SKU,
			"tenant A's analyze must not generate alerts for tenant B SKUs")
	}

	// Tenant B Analyze also succeeds independently.
	_, errB := repo.Analyze(testTenantB)
	require.Nil(t, errB, "Analyze for second tenant must also succeed")
}
