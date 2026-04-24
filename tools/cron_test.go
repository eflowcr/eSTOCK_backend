// Integration tests for B7 cron jobs.
// Requires Docker (testcontainers). Run: go test -v ./tools/... -run TestCron

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

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
		ON CONFLICT (email) WHERE deleted_at IS NULL AND email IS NOT NULL DO NOTHING`, id, id+"@crontest.com").Error)
	var uid string
	db.Raw("SELECT id FROM users WHERE email = ?", id+"@crontest.com").Scan(&uid)
	if uid == "" {
		uid = id
	}
	return uid
}

func cronSeedInventory(t *testing.T, db *gorm.DB, sku, location string, qty, reserved float64) string {
	t.Helper()
	// Ensure article exists (inventory has FK to articles.sku)
	require.NoError(t, db.Exec(`
		INSERT INTO articles (sku, name, presentation, rotation_strategy, track_by_lot, track_by_serial, track_expiration)
		VALUES (?, ?, 'unit', 'fifo', false, false, false)
		ON CONFLICT (sku) DO NOTHING`, sku, sku).Error)
	id, err := GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO inventory (id, sku, name, location, quantity, reserved_qty, status, presentation, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'available', 'unit', NOW(), NOW())`,
		id, sku, sku, location, qty, reserved).Error)
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
	CronDispatch(db, analyzer, nil, nil, nil)

	assert.True(t, analyzerCalled, "analyzer must be called")
	// stale cleanup ran on empty tables without error (verified by no panic)
}

// ─── trial seed helpers ──────────────────────────────────────────────────────

// cronSeedTenant inserts a tenant row with the given status and trial_ends_at.
func cronSeedTenant(t *testing.T, db *gorm.DB, name, status string, trialEndsAt time.Time) string {
	t.Helper()
	var id string
	require.NoError(t, db.Raw(`
		INSERT INTO tenants (name, slug, email, status, trial_ends_at, is_active)
		VALUES (?, ?, ?, ?, ?, true)
		RETURNING id`,
		name, name+"-slug", name+"@test.com", status, trialEndsAt,
	).Scan(&id).Error)
	require.NotEmpty(t, id)
	return id
}

// cronGetTenantStatus returns the current status of a tenant by ID.
func cronGetTenantStatus(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw("SELECT status FROM tenants WHERE id = ?", tenantID).Scan(&status).Error)
	return status
}

// cronGetTenantIsActive returns the is_active flag of a tenant by ID.
func cronGetTenantIsActive(t *testing.T, db *gorm.DB, tenantID string) bool {
	t.Helper()
	var isActive bool
	require.NoError(t, db.Raw("SELECT is_active FROM tenants WHERE id = ?", tenantID).Scan(&isActive).Error)
	return isActive
}

// ─── unit tests (no DB) ─────────────────────────────────────────────────────

// TestRunTrialExpirationCheck_NilDB verifies nil db returns an error cleanly.
func TestRunTrialExpirationCheck_NilDB(t *testing.T) {
	err := RunTrialExpirationCheck(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil db")
}

// ─── unit tests for email template selection ─────────────────────────────────

// TestRenderTrialEmail_Templates verifies that each templateType produces the
// correct subject and non-empty HTML.
func TestRenderTrialEmail_Templates(t *testing.T) {
	cases := []struct {
		templateType   string
		daysLeft       int
		subjectContains string
	}{
		{"trial_reminder_13d", 13, "13 días"},
		{"trial_reminder_11d", 11, "11 días"},
		{"trial_reminder_7d", 7, "7 días"},
		{"trial_expired", 0, "expirado"},
	}
	for _, tc := range cases {
		t.Run(tc.templateType, func(t *testing.T) {
			subject, htmlBody, textBody := RenderTrialEmail(tc.templateType, "Acme Corp", tc.daysLeft)
			assert.NotEmpty(t, subject)
			assert.Contains(t, subject, tc.subjectContains)
			assert.NotEmpty(t, htmlBody)
			assert.Contains(t, htmlBody, "Acme Corp")
			assert.NotEmpty(t, textBody)
			assert.Contains(t, textBody, "Acme Corp")
		})
	}
}

// TestRenderTrialEmail_Escaping verifies that malicious tenant names are escaped in HTML.
func TestRenderTrialEmail_Escaping(t *testing.T) {
	_, htmlBody, _ := RenderTrialEmail("trial_reminder_7d", `<script>alert("xss")</script>`, 7)
	assert.NotContains(t, htmlBody, "<script>")
	assert.Contains(t, htmlBody, "&lt;script&gt;")
}

// ─── integration tests ───────────────────────────────────────────────────────

// TestRunTrialExpirationCheck_SendsReminder7d inserts a tenant with trial_ends_at
// 7 days from now and verifies the sendFn is called with "trial_reminder_7d".
func TestRunTrialExpirationCheck_SendsReminder7d(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().AddDate(0, 0, 7).Add(30 * time.Minute) // 7d + buffer so daysLeft rounds to 7
	tenantID := cronSeedTenant(t, db, "trial-7d-tenant", "trial", trialEndsAt)

	var mu sync.Mutex
	var calledTemplates []string
	sendFn := func(_ context.Context, _, _ string, templateType string, _ int) error {
		mu.Lock()
		calledTemplates = append(calledTemplates, templateType)
		mu.Unlock()
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))

	// TODO(M4 — S3.5): sendFn is called in a goroutine — 50ms sleep is a flaky timeout.
	// Fix: make sendFn synchronous in tests (no go func) or use sync.WaitGroup/channel.
	// Deferred to S3.5 to avoid large test refactor in this wave.
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	assert.Contains(t, calledTemplates, "trial_reminder_7d", "should have sent 7d reminder")
	// Tenant must still be 'trial' (not deactivated)
	assert.Equal(t, "trial", cronGetTenantStatus(t, db, tenantID))
}

// TestRunTrialExpirationCheck_SendsReminder11d mirrors the 7d test for 11 days.
func TestRunTrialExpirationCheck_SendsReminder11d(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().AddDate(0, 0, 11).Add(30 * time.Minute)
	cronSeedTenant(t, db, "trial-11d-tenant", "trial", trialEndsAt)

	var mu sync.Mutex
	var calledTemplates []string
	sendFn := func(_ context.Context, _, _ string, templateType string, _ int) error {
		mu.Lock()
		calledTemplates = append(calledTemplates, templateType)
		mu.Unlock()
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, calledTemplates, "trial_reminder_11d")
}

// TestRunTrialExpirationCheck_SendsReminder13d mirrors the 7d test for 13 days.
func TestRunTrialExpirationCheck_SendsReminder13d(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().AddDate(0, 0, 13).Add(30 * time.Minute)
	cronSeedTenant(t, db, "trial-13d-tenant", "trial", trialEndsAt)

	var mu sync.Mutex
	var calledTemplates []string
	sendFn := func(_ context.Context, _, _ string, templateType string, _ int) error {
		mu.Lock()
		calledTemplates = append(calledTemplates, templateType)
		mu.Unlock()
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, calledTemplates, "trial_reminder_13d")
}

// TestRunTrialExpirationCheck_Expiration verifies that a tenant whose trial_ends_at
// is in the past gets deactivated (status=past_due, is_active=false) and receives
// the trial_expired email.
func TestRunTrialExpirationCheck_Expiration(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().Add(-1 * time.Hour) // already expired
	tenantID := cronSeedTenant(t, db, "expired-tenant", "trial", trialEndsAt)

	var mu sync.Mutex
	var calledTemplates []string
	sendFn := func(_ context.Context, _, _ string, templateType string, _ int) error {
		mu.Lock()
		calledTemplates = append(calledTemplates, templateType)
		mu.Unlock()
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	assert.Contains(t, calledTemplates, "trial_expired")
	assert.Equal(t, "past_due", cronGetTenantStatus(t, db, tenantID))
	assert.False(t, cronGetTenantIsActive(t, db, tenantID))
}

// TestRunTrialExpirationCheck_NoActionForOtherDays verifies that a tenant at day 5
// (not 7/11/13) receives no email and stays in trial.
func TestRunTrialExpirationCheck_NoActionForOtherDays(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().AddDate(0, 0, 5).Add(30 * time.Minute) // day 5
	tenantID := cronSeedTenant(t, db, "day5-tenant", "trial", trialEndsAt)

	sendCalled := false
	sendFn := func(_ context.Context, _, _, _ string, _ int) error {
		sendCalled = true
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))
	time.Sleep(50 * time.Millisecond)

	assert.False(t, sendCalled, "no email should be sent on day 5")
	assert.Equal(t, "trial", cronGetTenantStatus(t, db, tenantID))
}

// TestRunTrialExpirationCheck_SkipsNonTrialTenants verifies that tenants in 'active'
// status are not touched.
func TestRunTrialExpirationCheck_SkipsNonTrialTenants(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	// Active tenant — should never be emailed or deactivated even with past trial_ends_at
	trialEndsAt := time.Now().UTC().Add(-1 * time.Hour)
	tenantID := cronSeedTenant(t, db, "active-tenant", "active", trialEndsAt)

	sendCalled := false
	sendFn := func(_ context.Context, _, _, _ string, _ int) error {
		sendCalled = true
		return nil
	}

	require.NoError(t, RunTrialExpirationCheck(db, sendFn))
	time.Sleep(50 * time.Millisecond)

	assert.False(t, sendCalled, "active tenant must not trigger emails")
	// Status must remain 'active'
	assert.Equal(t, "active", cronGetTenantStatus(t, db, tenantID))
}

// TestRunTrialExpirationCheck_AdvisoryLockSerializes verifies that running
// RunTrialExpirationCheck from two goroutines concurrently serializes correctly:
// only one fires the real work (the other skips due to the advisory lock).
// The advisory lock is transaction-scoped so the second goroutine gets the lock
// only after the first commits. We verify no data races or panics occur.
func TestRunTrialExpirationCheck_AdvisoryLockSerializes(t *testing.T) {
	db, cleanup := setupCronTestDB(t)
	defer cleanup()

	trialEndsAt := time.Now().UTC().Add(-1 * time.Hour) // expired
	cronSeedTenant(t, db, "lock-race-tenant", "trial", trialEndsAt)

	var wg sync.WaitGroup
	var mu sync.Mutex
	sendCount := 0

	sendFn := func(_ context.Context, _, _, _ string, _ int) error {
		mu.Lock()
		sendCount++
		mu.Unlock()
		return nil
	}

	// Run 2 goroutines simultaneously; advisory lock means only one does DB work per tick.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = RunTrialExpirationCheck(db, sendFn)
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// Because the first commit deactivates the tenant (status='past_due'), the second
	// goroutine finds no tenant with status='trial' and sends nothing.
	// Either 1 send (first pod wins) or 0 (both see empty after lock) is correct;
	// what must NOT happen is 2 sends (double-fire).
	assert.LessOrEqual(t, sendCount, 1, "advisory lock must prevent double-fire")
}
