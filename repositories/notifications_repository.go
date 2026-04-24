package repositories

import (
	"errors"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationsRepository struct {
	DB *gorm.DB
}

var _ ports.NotificationsRepository = (*NotificationsRepository)(nil)

func (r *NotificationsRepository) Create(n *database.Notification) *responses.InternalResponse {
	id, err := tools.GenerateNanoid(r.DB)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error generando ID de notificación", Handled: false}
	}
	n.ID = id

	if err := r.DB.Create(n).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear notificación", Handled: false}
	}
	return nil
}

func (r *NotificationsRepository) ListByUser(params ports.ListNotificationsParams) ([]database.Notification, int64, *responses.InternalResponse) {
	q := r.DB.Model(&database.Notification{}).
		Where("user_id = ? AND tenant_id = ?", params.UserID, params.TenantID)

	if params.Unread != nil {
		q = q.Where("is_read = ?", !*params.Unread)
	}
	if params.EventType != nil {
		q = q.Where("event_type = ?", *params.EventType)
	}
	if params.From != nil {
		q = q.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		q = q.Where("created_at <= ?", *params.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, &responses.InternalResponse{Error: err, Message: "Error contando notificaciones", Handled: false}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	var items []database.Notification
	if err := q.Order("created_at DESC").Limit(limit).Offset(params.Offset).Find(&items).Error; err != nil {
		return nil, 0, &responses.InternalResponse{Error: err, Message: "Error listando notificaciones", Handled: false}
	}
	if items == nil {
		items = []database.Notification{}
	}
	return items, total, nil
}

func (r *NotificationsRepository) MarkRead(id, userID string) *responses.InternalResponse {
	now := time.Now()
	result := r.DB.Model(&database.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		return &responses.InternalResponse{Error: result.Error, Message: "Error al marcar notificación como leída", Handled: false}
	}
	if result.RowsAffected == 0 {
		return &responses.InternalResponse{Message: "Notificación no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return nil
}

func (r *NotificationsRepository) MarkAllRead(userID, tenantID string) *responses.InternalResponse {
	now := time.Now()
	if err := r.DB.Model(&database.Notification{}).
		Where("user_id = ? AND tenant_id = ? AND is_read = false", userID, tenantID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al marcar todas las notificaciones como leídas", Handled: false}
	}
	return nil
}

func (r *NotificationsRepository) CountUnread(userID, tenantID string) (int64, *responses.InternalResponse) {
	var count int64
	if err := r.DB.Model(&database.Notification{}).
		Where("user_id = ? AND tenant_id = ? AND is_read = false", userID, tenantID).
		Count(&count).Error; err != nil {
		return 0, &responses.InternalResponse{Error: err, Message: "Error contando notificaciones no leídas", Handled: false}
	}
	return count, nil
}

func (r *NotificationsRepository) GetUserEmail(userID string) (string, *responses.InternalResponse) {
	var email string
	if err := r.DB.Table("users").Select("email").Where("id = ?", userID).Scan(&email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", &responses.InternalResponse{Message: "Usuario no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return "", &responses.InternalResponse{Error: err, Message: "Error buscando email de usuario", Handled: false}
	}
	return email, nil
}

func (r *NotificationsRepository) GetPreferences(userID string) (map[string]database.NotificationPreference, *responses.InternalResponse) {
	var prefs []database.NotificationPreference
	if err := r.DB.Where("user_id = ?", userID).Find(&prefs).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error obteniendo preferencias", Handled: false}
	}
	result := make(map[string]database.NotificationPreference, len(prefs))
	for _, p := range prefs {
		result[p.EventType] = p
	}
	return result, nil
}

func (r *NotificationsRepository) UpsertPreference(pref *database.NotificationPreference) *responses.InternalResponse {
	pref.UpdatedAt = time.Now()
	if err := r.DB.Clauses(clause.OnConflict{
		// M7: PK now includes tenant_id — ON CONFLICT must match the full composite PK.
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "event_type"}, {Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"in_app", "email", "push", "updated_at"}),
	}).Create(pref).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error guardando preferencia", Handled: false}
	}
	return nil
}

func (r *NotificationsRepository) ListPreferences(userID string) ([]database.NotificationPreference, *responses.InternalResponse) {
	var prefs []database.NotificationPreference
	if err := r.DB.Where("user_id = ?", userID).Order("event_type ASC").Find(&prefs).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listando preferencias", Handled: false}
	}
	if prefs == nil {
		prefs = []database.NotificationPreference{}
	}
	return prefs, nil
}

