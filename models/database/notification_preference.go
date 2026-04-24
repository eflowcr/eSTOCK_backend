package database

import "time"

type NotificationPreference struct {
	UserID    string    `gorm:"column:user_id;primaryKey" json:"user_id"`
	EventType string    `gorm:"column:event_type;primaryKey" json:"event_type"`
	TenantID  string    `gorm:"column:tenant_id;primaryKey" json:"tenant_id"`
	InApp     bool      `gorm:"column:in_app;default:true" json:"in_app"`
	Email     bool      `gorm:"column:email;default:true" json:"email"`
	Push      bool      `gorm:"column:push;default:false" json:"push"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}
