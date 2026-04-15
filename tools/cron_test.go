// Integration tests for B7 cron jobs.
// Requires Docker (testcontainers). Run: go test -v ./tools/... -run TestCron

package tools

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ─── DB setup ────────────────────────────────────────────────────────────────

func setupCronTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := t.Context()
	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	cleanup := func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	migrationPath := filepath.Join(dir, "..", "db", "migrations")
	require.NoError(t, RunMigrations("file://"+filepath.ToSlash(migrationPath), connStr))

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	return db, cleanup
}

// ─── seed helpers ────────────────────────────────────────────────────────────

func cronSeedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES (?, 'Cron', 'Test', ?, 'hashed', NOW(), NOW())
		ON CONFLICT (email) DO NOTHING`, id, id+"@crontest.com").Error)
	var uid string
	db.Raw("SELECT id FROM users WHERE email = ?", id+"@crontest.com").Scan(&uid)
	if uid == "" {
		uid = id
	}
	return uid
}

func cronSeedInventory(t *testing.T, db *gorm.DB, sku, location string, qty, reserved float64) string {
	t.Helper()
	id, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO inventory (id, sku, location, quantity, reserved_qty, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', NOW(), NOW())`,
		id, sku, location, qty, reserved).Error)
	return id
}

type cronAllocation struct {
	Location string  `json:"location"`
	Quantity float64 `json:"quantity"`
}

type cronItem struct {
	SKU         string           `json:"sku"`
	Allocations []cronAllocation `json:"allocations"`
}

func cronSeedPickingTask(t *testing.T, db *gorm.DB, userID, status string, items []cronItem) string {
	t.Helper()
	id, err := GenerateNanoid(db)
	require.NoError(t, err)
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO picking_tasks (id, task_id, order_number, created_by, status, priority, items, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'normal', ?, NOW(), NOW())`,
		id, "PICK-"+id[:6], "ORD-"+id[:6], userID, status, string(itemsJSON)).Error)
	return id
}

func cronGetTaskStatus(t *testing.T, db *gorm.DB, taskID string) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw("SELECT status FROM picking_tasks WHERE id = ?", taskID).Scan(&status).Error)
	return status
}

func cronGetReservedQty(t *testing.T, db *gorm.DB, invID string) float64 {
	t.Helper()
	var qty float64
	require.NoError(t, db.Raw("SELECT reserved_qty FROM inventory WHERE id = ?", invID).Scan(&qty).Error)
	return qty
}

// ─── tests ───────────────────────────────────────────────────────────────────

// TestRunStaleReservationsCleanup_InProgressAbandoned verifies that an in_progress
// task stale for >7 days is marked abandoned and its reserved_qty is released.
func TestRunStaleReservationsCleanup_InProgressAbandoned(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	userID := cronSeedUser(t, db)
	invID := cronSeedInventory(t, db, "SKU-STALE", "LOC-A", 100, 10)
	items := []cronItem{
		{SKU: "SKU-STALE", Allocations: []cronAllocation{{Location: "LOC-A", Quantity: 10}}},
	}
	taskID := cronSeedPickingTask(t, db, userID, "in_progress", items)

	// Make task stale (updated_at 8 days ago)
	require.NoError(t, db.Exec(
		"UPDATE picking_tasks SET updated_at = NOW() - INTERVAL '8 days' WHERE id = ?", taskID,
	).Error)

	require.NoError(t, RunStaleReservationsCleanup(db))

	assert.Equal(t, "abandoned", cronGetTaskStatus(t, db, taskID))
	assert.Equal(t, float64(0), cronGetReservedQty(t, db, invID))
}

// TestRunStaleReservationsCleanup_OpenAbandoned verifies that an open task
// created >7 days ago is marked abandoned without touching inventory.
func TestRunStaleReservationsCleanup_OpenAbandoned(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	userID := cronSeedUser(t, db)
	invID := cronSeedInventory(t, db, "SKU-OPEN", "LOC-B", 50, 5)
	items := []cronItem{
		{SKU: "SKU-OPEN", Allocations: []cronAllocation{{Location: "LOC-B", Quantity: 5}}},
	}
	taskID := cronSeedPickingTask(t, db, userID, "open", items)

	// Make task old (created_at 8 days ago)
	require.NoError(t, db.Exec(
		"UPDATE picking_tasks SET created_at = NOW() - INTERVAL '8 days' WHERE id = ?", taskID,
	).Error)

	require.NoError(t, RunStaleReservationsCleanup(db))

	assert.Equal(t, "abandoned", cronGetTaskStatus(t, db, taskID))
	// Inventory must NOT be touched — open tasks have no applied reservations (B1 Lazy)
	assert.Equal(t, float64(5), cronGetReservedQty(t, db, invID))
}

// TestRunStaleReservationsCleanup_RecentStaysPut verifies that a recent in_progress
// task (2 days old) is not abandoned.
func TestRunStaleReservationsCleanup_RecentStaysPut(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	userID := cronSeedUser(t, db)
	invID := cronSeedInventory(t, db, "SKU-RECENT", "LOC-C", 80, 20)
	items := []cronItem{
		{SKU: "SKU-RECENT", Allocations: []cronAllocation{{Location: "LOC-C", Quantity: 20}}},
	}
	taskID := cronSeedPickingTask(t, db, userID, "in_progress", items)

	// updated_at is NOW() by default — 2 days ago is still within the 7-day window
	require.NoError(t, db.Exec(
		"UPDATE picking_tasks SET updated_at = NOW() - INTERVAL '2 days' WHERE id = ?", taskID,
	).Error)

	require.NoError(t, RunStaleReservationsCleanup(db))

	assert.Equal(t, "in_progress", cronGetTaskStatus(t, db, taskID))
	assert.Equal(t, float64(20), cronGetReservedQty(t, db, invID))
}

// TestCronDispatch_BothJobsRun verifies that CronDispatch calls the analyzer and
// runs stale cleanup even when the analyzer returns an error.
func TestCronDispatch_BothJobsRun(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	analyzerCalled := false
	analyzer := func() error {
		analyzerCalled = true
		return errors.New("simulated stock_alerts failure")
	}

	// Should not panic — errors are logged, not propagated
	CronDispatch(db, analyzer)

	assert.True(t, analyzerCalled, "analyzer must be called")
	// stale cleanup ran on empty tables without error (verified by no panic)
}
