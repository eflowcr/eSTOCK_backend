package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IdempotencyKeysRepository implements ports.IdempotencyKeysRepository
// over GORM. See the port doc for semantics.
type IdempotencyKeysRepository struct {
	DB *gorm.DB
}

var _ ports.IdempotencyKeysRepository = (*IdempotencyKeysRepository)(nil)

func (r *IdempotencyKeysRepository) Lookup(ctx context.Context, key, userID string) (*database.IdempotencyKey, error) {
	var row database.IdempotencyKey
	err := r.DB.WithContext(ctx).
		Where("key = ? AND user_id = ? AND expires_at > ?", key, userID, time.Now()).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *IdempotencyKeysRepository) Store(ctx context.Context, entry *database.IdempotencyKey) error {
	// ON CONFLICT DO NOTHING — the first writer wins. A concurrent retry that
	// raced and completed first already populated the row; preserving that
	// row keeps the operator's observed response stable across replays.
	return r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(entry).Error
}

func (r *IdempotencyKeysRepository) SweepExpired(ctx context.Context) (int64, error) {
	res := r.DB.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&database.IdempotencyKey{})
	return res.RowsAffected, res.Error
}
