package services

import (
	"context"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockNotifRepo struct {
	created     []database.Notification
	preferences map[string]database.NotificationPreference
	emailByUser map[string]string
}

func newMockNotifRepo() *mockNotifRepo {
	return &mockNotifRepo{
		preferences: make(map[string]database.NotificationPreference),
		emailByUser: make(map[string]string),
	}
}

func (m *mockNotifRepo) Create(n *database.Notification) *responses.InternalResponse {
	n.ID = "test-id-" + n.EventType
	n.CreatedAt = time.Now()
	m.created = append(m.created, *n)
	return nil
}

func (m *mockNotifRepo) ListByUser(params ports.ListNotificationsParams) ([]database.Notification, int64, *responses.InternalResponse) {
	var out []database.Notification
	for _, n := range m.created {
		if n.UserID == params.UserID {
			if params.Unread != nil && *params.Unread && n.IsRead {
				continue
			}
			out = append(out, n)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockNotifRepo) MarkRead(id, userID string) *responses.InternalResponse {
	for i, n := range m.created {
		if n.ID == id && n.UserID == userID {
			m.created[i].IsRead = true
			now := time.Now()
			m.created[i].ReadAt = &now
			return nil
		}
	}
	return &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: 404}
}

func (m *mockNotifRepo) MarkAllRead(userID, tenantID string) *responses.InternalResponse {
	now := time.Now()
	for i, n := range m.created {
		if n.UserID == userID {
			m.created[i].IsRead = true
			m.created[i].ReadAt = &now
		}
	}
	return nil
}

func (m *mockNotifRepo) CountUnread(userID, tenantID string) (int64, *responses.InternalResponse) {
	var count int64
	for _, n := range m.created {
		if n.UserID == userID && !n.IsRead {
			count++
		}
	}
	return count, nil
}

func (m *mockNotifRepo) GetUserEmail(userID string) (string, *responses.InternalResponse) {
	email, ok := m.emailByUser[userID]
	if !ok {
		return "", nil
	}
	return email, nil
}

func (m *mockNotifRepo) GetPreferences(userID, tenantID string) (map[string]database.NotificationPreference, *responses.InternalResponse) {
	result := make(map[string]database.NotificationPreference)
	for k, v := range m.preferences {
		result[k] = v
	}
	return result, nil
}

func (m *mockNotifRepo) UpsertPreference(pref *database.NotificationPreference) *responses.InternalResponse {
	m.preferences[pref.EventType] = *pref
	return nil
}

func (m *mockNotifRepo) ListPreferences(userID, tenantID string) ([]database.NotificationPreference, *responses.InternalResponse) {
	var out []database.NotificationPreference
	for _, p := range m.preferences {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

// noopEmailSender records calls without sending.
type noopEmailSender struct {
	sendCalls int
}

func (n *noopEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error { return nil }
func (n *noopEmailSender) Send(_ context.Context, to, subject, htmlBody, textBody string) error {
	n.sendCalls++
	return nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestNotificationsService_SendDefaultPrefs(t *testing.T) {
	repo := newMockNotifRepo()
	repo.emailByUser["user1"] = "user1@test.com"
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	err := svc.Send(context.Background(), "user1", "task_assigned", "Test title", "Test body", "task", "t1")
	require.NoError(t, err)

	require.Len(t, repo.created, 1)
	n := repo.created[0]
	assert.Equal(t, "user1", n.UserID)
	assert.Equal(t, "task_assigned", n.EventType)
	assert.Equal(t, "Test title", n.Title)
	assert.Contains(t, n.Channels, "in_app")
	assert.Contains(t, n.Channels, "email")
	assert.False(t, n.IsRead)
}

func TestNotificationsService_SendEmailDisabledByPref(t *testing.T) {
	repo := newMockNotifRepo()
	repo.emailByUser["user2"] = "user2@test.com"
	repo.preferences["task_assigned"] = database.NotificationPreference{
		UserID:    "user2",
		EventType: "task_assigned",
		InApp:     true,
		Email:     false,
		Push:      false,
	}
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	err := svc.Send(context.Background(), "user2", "task_assigned", "No email", "body", "", "")
	require.NoError(t, err)

	require.Len(t, repo.created, 1)
	assert.Equal(t, "in_app", repo.created[0].Channels)
}

func TestNotificationsService_SendNoPrefsUsesDefaults(t *testing.T) {
	repo := newMockNotifRepo()
	repo.emailByUser["user3"] = "user3@test.com"
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	// No preferences set for user3 — should apply defaults (in_app=true, email=true)
	err := svc.Send(context.Background(), "user3", "low_stock", "Low stock", "body", "", "")
	require.NoError(t, err)

	require.Len(t, repo.created, 1)
	assert.Contains(t, repo.created[0].Channels, "in_app")
	assert.Contains(t, repo.created[0].Channels, "email")
}

func TestNotificationsService_CountUnreadAfterSend(t *testing.T) {
	repo := newMockNotifRepo()
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	_ = svc.Send(context.Background(), "userA", "lot_expiring_7d", "T1", "B1", "", "")
	_ = svc.Send(context.Background(), "userA", "lot_expiring_1d", "T2", "B2", "", "")

	count, resp := repo.CountUnread("userA", "tenant-1")
	require.Nil(t, resp)
	assert.Equal(t, int64(2), count)
}

func TestNotificationsService_MarkRead(t *testing.T) {
	repo := newMockNotifRepo()
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	_ = svc.Send(context.Background(), "userB", "task_completed", "Done", "body", "task", "xyz")
	require.Len(t, repo.created, 1)
	id := repo.created[0].ID

	resp := repo.MarkRead(id, "userB")
	require.Nil(t, resp)
	assert.True(t, repo.created[0].IsRead)

	count, _ := repo.CountUnread("userB", "tenant-1")
	assert.Equal(t, int64(0), count)
}

func TestNotificationsService_MarkAllRead(t *testing.T) {
	repo := newMockNotifRepo()
	emailSender := &noopEmailSender{}
	svc := NewNotificationsService(repo, emailSender, "tenant-1")

	_ = svc.Send(context.Background(), "userC", "task_assigned", "T1", "B", "", "")
	_ = svc.Send(context.Background(), "userC", "task_completed", "T2", "B", "", "")

	resp := repo.MarkAllRead("userC", "tenant-1")
	require.Nil(t, resp)

	count, _ := repo.CountUnread("userC", "tenant-1")
	assert.Equal(t, int64(0), count)
}
