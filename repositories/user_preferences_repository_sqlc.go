package repositories

import (
	"context"
	"errors"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
)

// UserPreferencesRepositorySQLC implements ports.UserPreferencesRepository using sqlc.
type UserPreferencesRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewUserPreferencesRepositorySQLC returns a user preferences repository backed by sqlc.
func NewUserPreferencesRepositorySQLC(queries *sqlc.Queries) *UserPreferencesRepositorySQLC {
	return &UserPreferencesRepositorySQLC{queries: queries}
}

var _ ports.UserPreferencesRepository = (*UserPreferencesRepositorySQLC)(nil)

func (r *UserPreferencesRepositorySQLC) GetUserPreferences(ctx context.Context, userID string) (*ports.PreferencesEntry, error) {
	row, err := r.queries.GetUserPreferences(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e := sqlcPrefToEntry(row)
	return &e, nil
}

func (r *UserPreferencesRepositorySQLC) GetOrCreateUserPreferences(ctx context.Context, userID string) (*ports.PreferencesEntry, error) {
	row, err := r.queries.GetOrCreateUserPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	e := sqlcPrefToEntry(row)
	return &e, nil
}

func (r *UserPreferencesRepositorySQLC) UpdateUserPreferences(ctx context.Context, arg ports.UpdatePreferencesParams) (*ports.PreferencesEntry, error) {
	row, err := r.queries.UpdateUserPreferences(ctx, sqlc.UpdateUserPreferencesParams{
		UserID:                 arg.UserID,
		Theme:                  arg.Theme,
		Language:               arg.Language,
		EmailNotifications:     arg.EmailNotifications,
		PushNotifications:      arg.PushNotifications,
		MarketingNotifications: arg.MarketingNotifications,
		ProfileVisibility:      arg.ProfileVisibility,
		DataSharing:            arg.DataSharing,
	})
	if err != nil {
		return nil, err
	}
	e := sqlcPrefToEntry(row)
	return &e, nil
}

func sqlcPrefToEntry(row sqlc.UserPreference) ports.PreferencesEntry {
	return ports.PreferencesEntry{
		Theme:                  row.Theme,
		Language:               row.Language,
		EmailNotifications:     row.EmailNotifications,
		PushNotifications:      row.PushNotifications,
		MarketingNotifications: row.MarketingNotifications,
		ProfileVisibility:      row.ProfileVisibility,
		DataSharing:            row.DataSharing,
	}
}
