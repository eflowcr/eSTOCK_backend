package ports

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

type ListNotificationsParams struct {
	UserID    string
	TenantID  string
	Unread    *bool
	EventType *string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type NotificationsRepository interface {
	Create(n *database.Notification) *responses.InternalResponse
	ListByUser(params ListNotificationsParams) ([]database.Notification, int64, *responses.InternalResponse)
	MarkRead(id, userID string) *responses.InternalResponse
	MarkAllRead(userID, tenantID string) *responses.InternalResponse
	CountUnread(userID, tenantID string) (int64, *responses.InternalResponse)
	GetUserEmail(userID string) (string, *responses.InternalResponse)
	// GetPreferences returns a userID+tenantID-scoped map of stored preferences.
	// tenantID scopes results to the current tenant after migration 000021 introduced a 3-column PK.
	GetPreferences(userID, tenantID string) (map[string]database.NotificationPreference, *responses.InternalResponse)
	UpsertPreference(pref *database.NotificationPreference) *responses.InternalResponse
	// ListPreferences returns all preferences for userID within tenantID.
	ListPreferences(userID, tenantID string) ([]database.NotificationPreference, *responses.InternalResponse)
}
