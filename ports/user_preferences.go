package ports

import "context"

// UserPreferencesRepository provides per-user preferences (theme, language, notifications, privacy).
type UserPreferencesRepository interface {
	GetUserPreferences(ctx context.Context, userID string) (*PreferencesEntry, error)
	GetOrCreateUserPreferences(ctx context.Context, userID string) (*PreferencesEntry, error)
	UpdateUserPreferences(ctx context.Context, arg UpdatePreferencesParams) (*PreferencesEntry, error)
}

// PreferencesEntry is a single user's preferences for API responses.
type PreferencesEntry struct {
	Theme                  string `json:"theme"`
	Language               string `json:"language"`
	EmailNotifications     bool   `json:"email_notifications"`
	PushNotifications      bool   `json:"push_notifications"`
	MarketingNotifications bool   `json:"marketing_notifications"`
	ProfileVisibility      string `json:"profile_visibility"`
	DataSharing            bool   `json:"data_sharing"`
}

// UpdatePreferencesParams is the input for updating preferences.
type UpdatePreferencesParams struct {
	UserID                 string
	Theme                  string
	Language               string
	EmailNotifications     bool
	PushNotifications      bool
	MarketingNotifications bool
	ProfileVisibility      string
	DataSharing            bool
}
