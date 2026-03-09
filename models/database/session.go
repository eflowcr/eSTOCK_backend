package database

import (
	"encoding/json"
	"time"
)

// Session is the token-based session (access + refresh). Table: sessions.
type Session struct {
	ID               string          `gorm:"column:id;primaryKey" json:"id"`
	UserID           string          `gorm:"column:user_id;not null" json:"user_id"`
	SessionTypeID    string          `gorm:"column:session_type_id;not null" json:"session_type_id"`
	TokenHash        string          `gorm:"column:token_hash;not null" json:"-"`
	RefreshTokenHash *string         `gorm:"column:refresh_token_hash" json:"-"`
	UserAgent        *string         `gorm:"column:user_agent" json:"user_agent"`
	ClientIP         *string         `gorm:"column:client_ip" json:"client_ip"`
	IPAddress        *string         `gorm:"column:ip_address" json:"ip_address"` // INET as string in Go
	DeviceInfo       json.RawMessage `gorm:"column:device_info;type:jsonb" json:"device_info"`
	IsActive         bool            `gorm:"column:is_active;default:true" json:"is_active"`
	ExpiresAt        time.Time       `gorm:"column:expires_at;not null" json:"expires_at"`
	LastActivityAt   *time.Time      `gorm:"column:last_activity_at" json:"last_activity_at"`
	CreatedAt        time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedBy        *string         `gorm:"column:updated_by" json:"-"`
	UpdatedAt        *time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt        *time.Time      `gorm:"column:deleted_at" json:"-"`
}

func (Session) TableName() string {
	return "sessions"
}

// SessionType defines session kind (web, mobile, api, admin). Table: session_types.
type SessionType struct {
	ID               string     `gorm:"column:id;primaryKey" json:"id"`
	Name             string     `gorm:"column:name;not null" json:"name"`
	Description      *string    `gorm:"column:description" json:"description"`
	DurationMinutes  int        `gorm:"column:duration_minutes;not null" json:"duration_minutes"`
	IsActive         bool       `gorm:"column:is_active" json:"is_active"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedBy        *string    `gorm:"column:updated_by" json:"-"`
	UpdatedAt        *time.Time `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt        *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (SessionType) TableName() string {
	return "session_types"
}
