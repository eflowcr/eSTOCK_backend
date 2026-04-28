package database

import "time"

type Notification struct {
	ID           string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID     string     `gorm:"column:tenant_id" json:"tenant_id"`
	UserID       string     `gorm:"column:user_id" json:"user_id"`
	EventType    string     `gorm:"column:event_type" json:"event_type"`
	Title        string     `gorm:"column:title" json:"title"`
	Body         *string    `gorm:"column:body" json:"body,omitempty"`
	ResourceType *string    `gorm:"column:resource_type" json:"resource_type,omitempty"`
	ResourceID   *string    `gorm:"column:resource_id" json:"resource_id,omitempty"`
	Channels     string     `gorm:"column:channels;default:in_app" json:"channels"`
	IsRead       bool       `gorm:"column:is_read;default:false" json:"is_read"`
	ReadAt       *time.Time `gorm:"column:read_at" json:"read_at,omitempty"`
	SentEmailAt  *time.Time `gorm:"column:sent_email_at" json:"sent_email_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}
