// Integration test for B2a lot upsert — verifies ON CONFLICT consolidates quantity.
// Requires Docker (testcontainers). Run: go test -v ./repositories/... -run TestUpsertLot

package repositories

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupGORMTestDB(t *testing.T) (*gorm.DB, func()) {
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
	require.NoError(t, tools.RunMigrations("file://"+filepath.ToSlash(migrationPath), connStr))

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	return db, cleanup
}

// TestUpsertLot_SameLotNumberConsolidates verifies that receiving the same
// SKU+lot_number twice accumulates quantity instead of inserting a duplicate row.
func TestUpsertLot_SameLotNumberConsolidates(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	sku := "TEST-SKU-UPSERT"
	lotNumber := "LOT-A"

	upsertLot := func(qty float64) {
		lotID, err := tools.GenerateNanoid(db)
		require.NoError(t, err)
		err = db.Exec(`
			INSERT INTO lots (id, sku, lot_number, quantity, expiration_date, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, NULL, 'active', NOW(), NOW())
			ON CONFLICT (sku, lot_number) WHERE (status IS NULL OR status != 'archived')
			DO UPDATE SET
				quantity   = lots.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, lotID, sku, lotNumber, qty).Error
		require.NoError(t, err)
	}

	// First reception: 100 units
	upsertLot(100)

	// Second reception of same lot: 50 more
	upsertLot(50)

	// Should have exactly one row with consolidated quantity 150
	var count int64
	require.NoError(t, db.Table("lots").Where("sku = ? AND lot_number = ?", sku, lotNumber).Count(&count).Error)
	assert.Equal(t, int64(1), count, "expected exactly one lot row")

	var qty float64
	row := db.Raw("SELECT quantity FROM lots WHERE sku = ? AND lot_number = ?", sku, lotNumber).Row()
	require.NoError(t, row.Scan(&qty))
	assert.Equal(t, 150.0, qty, "expected consolidated quantity of 150")
}

// TestUpsertLot_ArchivedLotDoesNotConflict verifies that an archived lot
// does not trigger the ON CONFLICT clause — a new active lot is created instead.
func TestUpsertLot_ArchivedLotDoesNotConflict(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	sku := "TEST-SKU-ARCHIVED"
	lotNumber := "LOT-ARCH"

	// Insert an archived lot directly
	archivedID, _ := tools.GenerateNanoid(db)
	require.NoError(t, db.Exec(`
		INSERT INTO lots (id, sku, lot_number, quantity, expiration_date, status, created_at, updated_at)
		VALUES (?, ?, ?, 100, NULL, 'archived', NOW(), NOW())
	`, archivedID, sku, lotNumber).Error)

	// Now upsert same lot_number — should create a new active row, not conflict
	newID, _ := tools.GenerateNanoid(db)
	require.NoError(t, db.Exec(`
		INSERT INTO lots (id, sku, lot_number, quantity, expiration_date, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, NULL, 'active', NOW(), NOW())
		ON CONFLICT (sku, lot_number) WHERE (status IS NULL OR status != 'archived')
		DO UPDATE SET
			quantity   = lots.quantity + EXCLUDED.quantity,
			updated_at = NOW()
	`, newID, sku, lotNumber, 75).Error)

	// Should have two rows: one archived (qty 100) and one active (qty 75)
	var count int64
	require.NoError(t, db.Table("lots").Where("sku = ? AND lot_number = ?", sku, lotNumber).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	var activeQty float64
	row := db.Raw("SELECT quantity FROM lots WHERE sku = ? AND lot_number = ? AND status = 'active'", sku, lotNumber).Row()
	require.NoError(t, row.Scan(&activeQty))
	assert.Equal(t, 75.0, activeQty)
}
