package controllers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type NotificationsController struct {
	repo     ports.NotificationsRepository
	tenantID string
}

func NewNotificationsController(repo ports.NotificationsRepository, tenantID string) *NotificationsController {
	return &NotificationsController{repo: repo, tenantID: tenantID}
}

func (c *NotificationsController) List(ctx *gin.Context) {
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, ok := userID.(string)
	if !ok || uid == "" {
		tools.ResponseBadRequest(ctx, "ListNotifications", "Usuario no autenticado", "list_notifications")
		return
	}

	params := ports.ListNotificationsParams{
		UserID:   uid,
		TenantID: c.tenantID,
		Limit:    20,
	}

	if q := ctx.Query("unread"); q == "1" || q == "true" {
		v := true
		params.Unread = &v
	}
	if et := ctx.Query("event_type"); et != "" {
		params.EventType = &et
	}
	if lim := ctx.Query("limit"); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil && n > 0 && n <= 100 {
			params.Limit = n
		}
	}
	if off := ctx.Query("offset"); off != "" {
		if n, err := strconv.Atoi(off); err == nil && n >= 0 {
			params.Offset = n
		}
	}
	if from := ctx.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			params.From = &t
		}
	}
	if to := ctx.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			params.To = &t
		}
	}

	items, total, resp := c.repo.ListByUser(params)
	if resp != nil {
		writeErrorResponse(ctx, "ListNotifications", "list_notifications", resp)
		return
	}
	tools.ResponseOK(ctx, "ListNotifications", "Notificaciones obtenidas", "list_notifications",
		gin.H{"items": items, "total": total}, false, "")
}

func (c *NotificationsController) MarkRead(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MarkNotificationRead", "mark_notification_read", "ID de notificación inválido")
	if !ok {
		return
	}
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, _ := userID.(string)

	if resp := c.repo.MarkRead(id, uid); resp != nil {
		writeErrorResponse(ctx, "MarkNotificationRead", "mark_notification_read", resp)
		return
	}
	tools.ResponseOK(ctx, "MarkNotificationRead", "Notificación marcada como leída", "mark_notification_read", nil, false, "")
}

func (c *NotificationsController) MarkAllRead(ctx *gin.Context) {
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, _ := userID.(string)

	if resp := c.repo.MarkAllRead(uid, c.tenantID); resp != nil {
		writeErrorResponse(ctx, "MarkAllNotificationsRead", "mark_all_notifications_read", resp)
		return
	}
	tools.ResponseOK(ctx, "MarkAllNotificationsRead", "Todas las notificaciones marcadas como leídas", "mark_all_notifications_read", nil, false, "")
}

func (c *NotificationsController) CountUnread(ctx *gin.Context) {
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, _ := userID.(string)

	count, resp := c.repo.CountUnread(uid, c.tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "CountUnreadNotifications", "count_unread", resp)
		return
	}
	tools.ResponseOK(ctx, "CountUnreadNotifications", "Conteo de no leídas", "count_unread",
		gin.H{"count": count}, false, "")
}

func (c *NotificationsController) GetPreferences(ctx *gin.Context) {
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, _ := userID.(string)

	prefs, resp := c.repo.ListPreferences(uid, c.tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "GetNotificationPreferences", "get_notification_preferences", resp)
		return
	}
	tools.ResponseOK(ctx, "GetNotificationPreferences", "Preferencias obtenidas", "get_notification_preferences", prefs, false, "")
}

func (c *NotificationsController) UpsertPreferences(ctx *gin.Context) {
	userID, _ := ctx.Get(tools.ContextKeyUserID)
	uid, _ := userID.(string)

	var body []struct {
		EventType string `json:"event_type" binding:"required"`
		InApp     *bool  `json:"in_app"`
		Email     *bool  `json:"email"`
		Push      *bool  `json:"push"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpsertNotificationPreferences", "Datos inválidos", "upsert_notification_preferences")
		return
	}

	validEventTypes := map[string]bool{
		"task_assigned": true, "task_completed": true,
		"lot_expiring_7d": true, "lot_expiring_1d": true,
		"low_stock": true, "user_welcome": true,
	}

	for _, item := range body {
		if !validEventTypes[item.EventType] {
			writeErrorResponse(ctx, "UpsertNotificationPreferences", "upsert_notification_preferences",
				&responses.InternalResponse{
					Message:    fmt.Sprintf("event_type inválido: %s", item.EventType),
					Handled:    true,
					StatusCode: responses.StatusBadRequest,
				})
			return
		}

		pref := &database.NotificationPreference{
			UserID:    uid,
			EventType: item.EventType,
			TenantID:  c.tenantID,
			InApp:     true,
			Email:     true,
			Push:      false,
		}
		if item.InApp != nil {
			pref.InApp = *item.InApp
		}
		if item.Email != nil {
			pref.Email = *item.Email
		}
		if item.Push != nil {
			pref.Push = *item.Push
		}

		if resp := c.repo.UpsertPreference(pref); resp != nil {
			writeErrorResponse(ctx, "UpsertNotificationPreferences", "upsert_notification_preferences", resp)
			return
		}
	}

	tools.ResponseOK(ctx, "UpsertNotificationPreferences", "Preferencias guardadas", "upsert_notification_preferences", nil, false, "")
}

