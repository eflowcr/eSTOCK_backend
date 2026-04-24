// Tests for HR-S3.5 C3 fix: per-tenant iteration in RunLotExpirationCheck and
// RunLowStockNotifications. Pre-W5.5 these helpers queried lots/stock_alerts
// globally and dispatched notifications via a single closure, so a tenant 2
// expiring lot would email tenant 1 admins. Now both helpers iterate active
// tenants and pass tenant_id into notifyFn so the caller can scope recipients.

package tools

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunLotExpirationCheck_PerTenantIteration is an integration test that:
//  1. seeds 2 tenants
//  2. seeds an expiring lot for each tenant
//  3. calls RunLotExpirationCheck and asserts notifyFn was invoked per tenant,
//     with the tenant_id matching the lot's tenant — never crossing tenants.
func TestRunLotExpirationCheck_PerTenantIteration(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	// Seed two tenants.
	tenantA := cronSeedTenant(t, db, "tenant-a", "active", time.Now().UTC().AddDate(1, 0, 0))
	tenantB := cronSeedTenant(t, db, "tenant-b", "active", time.Now().UTC().AddDate(1, 0, 0))

	// Seed one expiring lot per tenant. lots requires an article; reuse cronSeedInventory
	// which creates the article row, then INSERT lots directly.
	cronSeedInventory(t, db, "SKU-EXP-A", "LOC-1", 100, 0)
	cronSeedInventory(t, db, "SKU-EXP-B", "LOC-1", 100, 0)

	exp := time.Now().UTC().AddDate(0, 0, 5) // 5 days out → in window

	idA, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO lots (id, tenant_id, lot_number, sku, expiration_date, quantity, created_at, updated_at)
		VALUES (?, ?::uuid, 'LOT-A-001', 'SKU-EXP-A', ?, 50, NOW(), NOW())`,
		idA, tenantA, exp).Error)

	idB, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO lots (id, tenant_id, lot_number, sku, expiration_date, quantity, created_at, updated_at)
		VALUES (?, ?::uuid, 'LOT-B-001', 'SKU-EXP-B', ?, 50, NOW(), NOW())`,
		idB, tenantB, exp).Error)

	// notifyFn records which tenant + lot was notified.
	var mu sync.Mutex
	type call struct{ tenantID, eventType, title string }
	var calls []call
	notifyFn := func(tenantID, eventType, title, body string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call{tenantID, eventType, title})
		return nil
	}

	require.NoError(t, RunLotExpirationCheck(db, notifyFn))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 2, "must notify exactly once per tenant — see HR-S3.5 C3")

	// Verify each call's tenant matches the lot's tenant: no cross-tenant leak.
	tenantsSeen := map[string]bool{}
	for _, c := range calls {
		tenantsSeen[c.tenantID] = true
		switch c.tenantID {
		case tenantA:
			assert.Contains(t, c.title, "LOT-A-001", "tenant A must receive its OWN lot notification")
		case tenantB:
			assert.Contains(t, c.title, "LOT-B-001", "tenant B must receive its OWN lot notification")
		default:
			t.Errorf("unexpected tenant in notify call: %s", c.tenantID)
		}
	}
	assert.True(t, tenantsSeen[tenantA], "tenant A must be notified")
	assert.True(t, tenantsSeen[tenantB], "tenant B must be notified")
}

// TestRunLowStockNotifications_PerTenantIteration mirrors the lot test for the
// low-stock alert helper.
func TestRunLowStockNotifications_PerTenantIteration(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	tenantA := cronSeedTenant(t, db, "alert-tenant-a", "active", time.Now().UTC().AddDate(1, 0, 0))
	tenantB := cronSeedTenant(t, db, "alert-tenant-b", "active", time.Now().UTC().AddDate(1, 0, 0))

	// Seed one unresolved low_stock alert per tenant.
	idA, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO stock_alerts (id, tenant_id, sku, alert_type, message, is_resolved, created_at, updated_at)
		VALUES (?, ?::uuid, 'SKU-LOW-A', 'low_stock', 'tenant A low stock', false, NOW(), NOW())`,
		idA, tenantA).Error)

	idB, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO stock_alerts (id, tenant_id, sku, alert_type, message, is_resolved, created_at, updated_at)
		VALUES (?, ?::uuid, 'SKU-LOW-B', 'low_stock', 'tenant B low stock', false, NOW(), NOW())`,
		idB, tenantB).Error)

	var mu sync.Mutex
	type call struct{ tenantID, sku, message string }
	var calls []call
	notifyFn := func(tenantID, sku, message string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call{tenantID, sku, message})
		return nil
	}

	require.NoError(t, RunLowStockNotifications(db, notifyFn))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 2, "must notify exactly once per tenant — HR-S3.5 C3")

	for _, c := range calls {
		switch c.tenantID {
		case tenantA:
			assert.Equal(t, "SKU-LOW-A", c.sku)
			assert.Contains(t, c.message, "tenant A")
		case tenantB:
			assert.Equal(t, "SKU-LOW-B", c.sku)
			assert.Contains(t, c.message, "tenant B")
		default:
			t.Errorf("unexpected tenant in notify call: %s", c.tenantID)
		}
	}
}

// TestRunLotExpirationCheck_PerTenantErrorContinues verifies the resilience
// contract: when notifyFn returns an error for one tenant, the next tenant still
// gets processed (no all-or-nothing failure).
func TestRunLotExpirationCheck_PerTenantErrorContinues(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	tenantA := cronSeedTenant(t, db, "err-tenant-a", "active", time.Now().UTC().AddDate(1, 0, 0))
	tenantB := cronSeedTenant(t, db, "err-tenant-b", "active", time.Now().UTC().AddDate(1, 0, 0))

	cronSeedInventory(t, db, "SKU-ERR-A", "LOC-1", 100, 0)
	cronSeedInventory(t, db, "SKU-ERR-B", "LOC-1", 100, 0)
	exp := time.Now().UTC().AddDate(0, 0, 5)

	for _, x := range []struct{ tid, sku string }{{tenantA, "SKU-ERR-A"}, {tenantB, "SKU-ERR-B"}} {
		id, err := GenerateNanoid(db)
		require.NoError(t, err)
		require.NoError(t, db.Exec(`
			INSERT INTO lots (id, tenant_id, lot_number, sku, expiration_date, quantity, created_at, updated_at)
			VALUES (?, ?::uuid, ?, ?, ?, 50, NOW(), NOW())`,
			id, x.tid, "LOT-"+x.sku, x.sku, exp).Error)
	}

	var mu sync.Mutex
	tenantsCalled := map[string]int{}
	notifyFn := func(tenantID, eventType, title, body string) error {
		mu.Lock()
		defer mu.Unlock()
		tenantsCalled[tenantID]++
		// Simulate failure for tenant A — tenant B must still receive its notification.
		if tenantID == tenantA {
			return assert.AnError
		}
		return nil
	}

	require.NoError(t, RunLotExpirationCheck(db, notifyFn),
		"a per-tenant notify error must not propagate as a top-level failure")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, tenantsCalled[tenantA], "tenant A's notify was invoked once (and errored — logged)")
	assert.Equal(t, 1, tenantsCalled[tenantB], "tenant B must still receive notification despite tenant A's error")
}
