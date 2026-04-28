package database

import "time"

type PasswordResetToken struct {
	ID        string     `gorm:"column:id;primaryKey" json:"id"`
	UserID    string     `gorm:"column:user_id" json:"user_id"`
	Token     string     `gorm:"column:token" json:"token"`
	ExpiresAt time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
