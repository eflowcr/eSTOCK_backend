package database

import "time"

// SignupToken holds a pending self-service signup email verification token.
// Token is a crypto-random 32-byte hex string. UsedAt is set once the signup is confirmed.
type SignupToken struct {
	ID         string     `gorm:"column:id;primaryKey" json:"id"`
	Email      string     `gorm:"column:email" json:"email"`
	TenantName string     `gorm:"column:tenant_name" json:"tenant_name"`
	TenantSlug string     `gorm:"column:tenant_slug" json:"tenant_slug"`
	Token      string     `gorm:"column:token;unique" json:"-"` // sensitive — omit from JSON responses
	ExpiresAt  time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt     *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SignupToken) TableName() string {
	return "signup_tokens"
}
