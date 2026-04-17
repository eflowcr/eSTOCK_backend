package services

import (
	"context"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/rs/zerolog/log"
)

// defaultPrefs are applied when a user has no stored preference for an event type.
var defaultPrefs = database.NotificationPreference{
	InApp: true,
	Email: true,
	Push:  false,
}

// NotificationsService manages in-app and email notifications.
type NotificationsService struct {
	repo        ports.NotificationsRepository
	emailSender tools.EmailSender
	tenantID    string
}

func NewNotificationsService(repo ports.NotificationsRepository, emailSender tools.EmailSender, tenantID string) *NotificationsService {
	return &NotificationsService{
		repo:        repo,
		emailSender: emailSender,
		tenantID:    tenantID,
	}
}

// Send creates an in-app notification and optionally emails the user using their stored preferences
// (defaults: in_app=true, email=true, push=false). Email is fire-and-forget with a 5s timeout.
func (s *NotificationsService) Send(ctx context.Context, userID, eventType, title, body, resourceType, resourceID string) error {
	prefs, _ := s.repo.GetPreferences(userID)
	pref, hasPref := prefs[eventType]
	if !hasPref {
		pref = defaultPrefs
		pref.UserID = userID
		pref.EventType = eventType
		pref.TenantID = s.tenantID
	}

	var activeChannels []string
	if pref.InApp {
		activeChannels = append(activeChannels, "in_app")
	}
	emailEnabled := pref.Email && s.emailSender != nil
	if emailEnabled {
		activeChannels = append(activeChannels, "email")
	}
	if pref.Push {
		activeChannels = append(activeChannels, "push")
	}
	if len(activeChannels) == 0 {
		activeChannels = []string{"in_app"}
	}

	n := &database.Notification{
		TenantID:  s.tenantID,
		UserID:    userID,
		EventType: eventType,
		Title:     title,
		Channels:  strings.Join(activeChannels, ","),
	}
	if body != "" {
		n.Body = &body
	}
	if resourceType != "" {
		n.ResourceType = &resourceType
	}
	if resourceID != "" {
		n.ResourceID = &resourceID
	}

	if resp := s.repo.Create(n); resp != nil {
		return resp.Error
	}

	if emailEnabled {
		capturedBody := body
		capturedTitle := title
		capturedEvent := eventType
		capturedUser := userID
		go func() {
			timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			toEmail, resp := s.repo.GetUserEmail(capturedUser)
			if resp != nil || toEmail == "" {
				log.Warn().Str("user_id", capturedUser).Str("event", capturedEvent).Msg("notifications: could not get user email")
				return
			}

			htmlBody, textBody := tools.RenderNotificationEmail(capturedEvent, capturedTitle, capturedBody)
			if err := s.emailSender.Send(timeout, toEmail, capturedTitle, htmlBody, textBody); err != nil {
				log.Warn().Err(err).Str("user_id", capturedUser).Str("event", capturedEvent).Msg("notifications: email send failed")
			} else {
				log.Debug().Str("user_id", capturedUser).Str("event", capturedEvent).Msg("notifications: email sent")
			}
		}()
	}

	return nil
}

// GetPreferences returns stored preferences for a user.
func (s *NotificationsService) GetPreferences(userID string) ([]database.NotificationPreference, error) {
	prefs, resp := s.repo.ListPreferences(userID)
	if resp != nil {
		return nil, resp.Error
	}
	return prefs, nil
}

// UpsertPreference stores a user+event_type preference.
func (s *NotificationsService) UpsertPreference(pref *database.NotificationPreference) error {
	if pref.TenantID == "" {
		pref.TenantID = s.tenantID
	}
	if resp := s.repo.UpsertPreference(pref); resp != nil {
		return resp.Error
	}
	return nil
}
