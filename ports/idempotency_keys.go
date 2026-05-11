package ports

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/database"
)

// IdempotencyKeysRepository is the persistence contract for the mobile S7.2
// outbox replay dedup table. Middleware looks up by (key, user_id) and
// either returns the cached response or captures the fresh one after the
// handler runs.
type IdempotencyKeysRepository interface {
	// Lookup returns the cached row if it exists AND has not expired.
	// Returns (nil, nil) when there's no hit; (nil, err) only on real DB errors.
	Lookup(ctx context.Context, key, userID string) (*database.IdempotencyKey, error)

	// Store inserts the cached response. Uses ON CONFLICT DO NOTHING so a
	// concurrent duplicate request that completed first wins — the second
	// request's slightly-different response is dropped, which is correct
	// idempotency semantics (the operator sees the FIRST observable result).
	Store(ctx context.Context, entry *database.IdempotencyKey) error

	// SweepExpired deletes rows older than now. Called from a cron worker.
	// Returns the count of deleted rows for observability.
	SweepExpired(ctx context.Context) (int64, error)
}
