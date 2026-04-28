// Integration tests for InventoryCountsRepository (W0.5 fix-wave).
// Requires Docker (testcontainers). Runs against the real migrations + a fresh
// Postgres so the unique index from migration 000019 is in effect.
//
// Run from backend dir: go test -v ./repositories/... -run TestInventoryCountsRepository

package repositories

import (
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupCountsRepoEnv(t *testing.T) (*InventoryCountsRepository, *gorm.DB, func()) {
	t.Helper()
	connStr, cleanup := setupTestDB(t)
	runMigrations(t, connStr)

	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err, "failed to open gorm")

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	repo := &InventoryCountsRepository{DB: db}
	return repo, db, cleanup
}

// seedCountsFixtures seeds a user, a location, an article, an inventory row, and
// a draft inventory_count tied to the user. Returns the IDs callers need for the
// AddLine / SubmitWithAdjustments tests.
func seedCountsFixtures(t *testing.T, db *gorm.DB, sku, locCode string, expectedQty float64) (userID, locID, countID string) {
	t.Helper()

	// User — relies on the set_default_user_role trigger to populate role_id.
	userID = "user-test-1"
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, name, email, password, is_active) VALUES (?, ?, ?, ?, true)`,
		userID, "Test User", "test-"+sku+"@test.local", "x").Error, "seed user")

	// Location.
	locID = "loc-test-" + locCode
	require.NoError(t, db.Exec(
		`INSERT INTO locations (id, location_code, type, is_active, is_way_out) VALUES (?, ?, ?, true, false)`,
		locID, locCode, "BIN").Error, "seed location")

	// Article (track_by_lot / track_by_serial defaults to false). Note: the
	// articles schema may not have all the columns referenced; insert minimally.
	require.NoError(t, db.Exec(
		`INSERT INTO articles (sku, name, presentation) VALUES (?, ?, ?) ON CONFLICT (sku) DO NOTHING`,
		sku, "Test Article", "unit").Error, "seed article")

	// Inventory.
	require.NoError(t, db.Exec(
		`INSERT INTO inventory (id, sku, name, location, quantity, status, presentation) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"inv-test-"+sku, sku, "Test Article", locCode, expectedQty, "ok", "unit").Error, "seed inventory")

	// Count.
	countID = "cnt-test-" + sku
	require.NoError(t, db.Exec(
		`INSERT INTO inventory_counts (id, code, name, status, created_by) VALUES (?, ?, ?, ?, ?)`,
		countID, "CC-TEST-"+sku, "Test Count", "in_progress", userID).Error, "seed count")

	return userID, locID, countID
}

// W0.5 N1-3 regression — re-scanning the same (count, location, sku, lot, serial)
// must converge to a single line whose values are the latest scan's, not duplicate
// rows that would double-count variance at submit time.
func TestInventoryCountsRepository_AddLine_Idempotent(t *testing.T) {
	repo, db, cleanup := setupCountsRepoEnv(t)
	defer cleanup()

	userID, locID, countID := seedCountsFixtures(t, db, "SKU-IDEMPOTENT", "LOC-IDEM", 10)

	// First scan: qty=12, variance=2.
	first := &database.InventoryCountLine{
		CountID:     countID,
		LocationID:  locID,
		SKU:         "SKU-IDEMPOTENT",
		ExpectedQty: 10,
		ScannedQty:  12,
		VarianceQty: 2,
		ScannedBy:   userID,
	}
	require.Nil(t, repo.AddLine(first))

	// Second scan, same logical key, different qty — should UPSERT, not duplicate.
	second := &database.InventoryCountLine{
		CountID:     countID,
		LocationID:  locID,
		SKU:         "SKU-IDEMPOTENT",
		ExpectedQty: 10,
		ScannedQty:  9,
		VarianceQty: -1,
		ScannedBy:   userID,
	}
	require.Nil(t, repo.AddLine(second))

	var count int64
	require.NoError(t, db.Model(&database.InventoryCountLine{}).Where("count_id = ?", countID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "two scans of the same (count,location,sku) must yield exactly one row (last-scan-wins)")

	var stored database.InventoryCountLine
	require.NoError(t, db.Where("count_id = ?", countID).First(&stored).Error)
	assert.Equal(t, 9.0, stored.ScannedQty, "latest scan's quantity must overwrite the prior")
	assert.Equal(t, -1.0, stored.VarianceQty, "latest scan's variance must overwrite the prior")
}

// W0.5 N2-2 regression — variance is recomputed at submit time. Simulate a
// concurrent stock movement between scan and submit by mutating inventory.quantity
// after AddLine but before SubmitWithAdjustments. The persisted variance must
// reflect the latest expected_qty.
func TestInventoryCountsRepository_Submit_RecomputesVariance(t *testing.T) {
	repo, db, cleanup := setupCountsRepoEnv(t)
	defer cleanup()

	sku, locCode := "SKU-RECOMPUTE", "LOC-REC"
	userID, locID, countID := seedCountsFixtures(t, db, sku, locCode, 10)

	// Operator scans 10 → variance 0 at scan time.
	line := &database.InventoryCountLine{
		CountID:     countID,
		LocationID:  locID,
		SKU:         sku,
		ExpectedQty: 10,
		ScannedQty:  10,
		VarianceQty: 0,
		ScannedBy:   userID,
	}
	require.Nil(t, repo.AddLine(line))

	// Stock moves between scan and submit: picking pulls 3 units. Real stock is now 7.
	require.NoError(t, db.Exec(`UPDATE inventory SET quantity = 7 WHERE sku = ? AND location = ?`, sku, locCode).Error)

	// stubCreator captures the adjustments fan-out generates so we can assert on direction + qty.
	stub := &recomputeStubCreator{}
	resp := repo.SubmitWithAdjustments(countID, userID, stub)
	require.Nil(t, resp)

	// Line's variance must be recomputed to (10 scanned − 7 expected) = 3 inbound.
	var stored database.InventoryCountLine
	require.NoError(t, db.Where("id = ?", line.ID).First(&stored).Error)
	assert.Equal(t, 7.0, stored.ExpectedQty, "expected_qty must be re-read at submit time")
	assert.Equal(t, 3.0, stored.VarianceQty, "variance must be recomputed against live inventory")

	require.Len(t, stub.calls, 1)
	assert.Equal(t, "INVENTORY_COUNT_INBOUND", stub.calls[0].Reason)
	assert.Equal(t, 3.0, stub.calls[0].AdjustmentQuantity, "fan-out must use the recomputed variance, not the stale one")
}

// recomputeStubCreator is a minimal InventoryAdjustmentsCreator that records calls
// and inserts a real adjustments row in the caller's tx so the count's
// adjustment_id FK is satisfied at MarkSubmitted time. Does NOT mutate inventory
// — that's the real AdjustmentsService's job; this stub only proves the
// fan-out plumbing works.
type recomputeStubCreator struct {
	calls []requests.CreateAdjustment
	seq   int
}

func (s *recomputeStubCreator) CreateAdjustmentTx(tx *gorm.DB, userId string, adj requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	s.calls = append(s.calls, adj)
	s.seq++
	id := "adj-stub-" + adj.SKU + "-" + adj.Reason
	// W0.6 merge fix: dev sprint-s2 added CHECK constraint on adjustment_type
	// (must be 'increase' / 'decrease' / 'count_reconcile'). Counts always emit
	// count_reconcile per AdjustmentsService.CreateAdjustmentTx contract; mirror
	// that here so the stub satisfies the constraint.
	adjType := adj.AdjustmentType
	if adjType == "" {
		adjType = "count_reconcile"
	}
	row := database.Adjustment{
		ID:               id,
		SKU:              adj.SKU,
		Location:         adj.Location,
		PreviousQuantity: 0,
		AdjustmentQty:    int(adj.AdjustmentQuantity),
		NewQuantity:      0,
		Reason:           adj.Reason,
		Notes:            &adj.Notes,
		UserID:           userId,
		AdjustmentType:   adjType,
		CreatedAt:        time.Now(),
	}
	if err := tx.Table((database.Adjustment{}).TableName()).Create(&row).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: err.Error(), Handled: false}
	}
	return &row, nil
}

// Compile-time interface check.
var _ ports.InventoryAdjustmentsCreator = (*recomputeStubCreator)(nil)
