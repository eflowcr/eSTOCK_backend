package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── stub repo ───────────────────────────────────────────────────────────────

type stubNotifRepo struct {
	notifications []database.Notification
	prefs         []database.NotificationPreference
	tenantID      string
}

func (s *stubNotifRepo) Create(n *database.Notification) *responses.InternalResponse {
	n.ID = "notif-" + n.EventType
	s.notifications = append(s.notifications, *n)
	return nil
}

func (s *stubNotifRepo) ListByUser(params ports.ListNotificationsParams) ([]database.Notification, int64, *responses.InternalResponse) {
	var out []database.Notification
	for _, n := range s.notifications {
		if n.UserID == params.UserID && n.TenantID == params.TenantID {
			if params.Unread != nil && *params.Unread && n.IsRead {
				continue
			}
			out = append(out, n)
		}
	}
	return out, int64(len(out)), nil
}

func (s *stubNotifRepo) MarkRead(id, userID string) *responses.InternalResponse {
	for i, n := range s.notifications {
		if n.ID == id {
			if n.UserID != userID {
				return &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: 404}
			}
			s.notifications[i].IsRead = true
			return nil
		}
	}
	return &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: 404}
}

func (s *stubNotifRepo) MarkAllRead(userID, tenantID string) *responses.InternalResponse {
	now := time.Now()
	for i, n := range s.notifications {
		if n.UserID == userID {
			s.notifications[i].IsRead = true
			s.notifications[i].ReadAt = &now
		}
	}
	return nil
}

func (s *stubNotifRepo) CountUnread(userID, tenantID string) (int64, *responses.InternalResponse) {
	var count int64
	for _, n := range s.notifications {
		if n.UserID == userID && !n.IsRead {
			count++
		}
	}
	return count, nil
}

func (s *stubNotifRepo) GetUserEmail(userID string) (string, *responses.InternalResponse) {
	return "", nil
}

func (s *stubNotifRepo) GetPreferences(userID string) (map[string]database.NotificationPreference, *responses.InternalResponse) {
	return map[string]database.NotificationPreference{}, nil
}

func (s *stubNotifRepo) UpsertPreference(pref *database.NotificationPreference) *responses.InternalResponse {
	s.prefs = append(s.prefs, *pref)
	return nil
}

func (s *stubNotifRepo) ListPreferences(userID string) ([]database.NotificationPreference, *responses.InternalResponse) {
	return s.prefs, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func setupNotifRouter(repo *stubNotifRepo, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := NewNotificationsController(repo, "test-tenant")

	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Next()
	})

	r.GET("/notifications", ctrl.List)
	r.GET("/notifications/count", ctrl.CountUnread)
	r.PATCH("/notifications/mark-all-read", ctrl.MarkAllRead)
	r.PATCH("/notifications/:id/read", ctrl.MarkRead)
	r.GET("/notifications/preferences", ctrl.GetPreferences)
	r.PUT("/notifications/preferences", ctrl.UpsertPreferences)
	return r
}

func seedNotif(repo *stubNotifRepo, userID, tenantID, eventType string, isRead bool) {
	repo.notifications = append(repo.notifications, database.Notification{
		ID:        "id-" + eventType,
		UserID:    userID,
		TenantID:  tenantID,
		EventType: eventType,
		Title:     "title-" + eventType,
		Channels:  "in_app",
		IsRead:    isRead,
		CreatedAt: time.Now(),
	})
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestNotificationsController_ListOnlyOwn(t *testing.T) {
	repo := &stubNotifRepo{}
	seedNotif(repo, "user1", "test-tenant", "task_assigned", false)
	seedNotif(repo, "user2", "test-tenant", "task_completed", false)

	r := setupNotifRouter(repo, "user1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	items := data["items"].([]interface{})
	assert.Len(t, items, 1) // only user1's notification
}

func TestNotificationsController_MarkRead_NotOwnNotif_Returns404(t *testing.T) {
	repo := &stubNotifRepo{}
	// seed a notification owned by user2
	repo.notifications = append(repo.notifications, database.Notification{
		ID:       "other-notif",
		UserID:   "user2",
		TenantID: "test-tenant",
		IsRead:   false,
	})

	r := setupNotifRouter(repo, "user1") // authenticated as user1
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/notifications/other-notif/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotificationsController_CountUnread(t *testing.T) {
	repo := &stubNotifRepo{}
	seedNotif(repo, "user1", "test-tenant", "task_assigned", false)
	seedNotif(repo, "user1", "test-tenant", "task_completed", true) // already read

	r := setupNotifRouter(repo, "user1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications/count", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["count"])
}

func TestNotificationsController_UpsertPreferences_InvalidEventType(t *testing.T) {
	repo := &stubNotifRepo{}
	r := setupNotifRouter(repo, "user1")

	payload := `[{"event_type":"non_existent_event","in_app":true}]`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/preferences", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationsController_UpsertPreferences_ValidEventType(t *testing.T) {
	repo := &stubNotifRepo{}
	r := setupNotifRouter(repo, "user1")

	payload := `[{"event_type":"task_assigned","in_app":true,"email":false}]`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/preferences", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.prefs, 1)
	assert.Equal(t, "task_assigned", repo.prefs[0].EventType)
	assert.False(t, repo.prefs[0].Email)
}
