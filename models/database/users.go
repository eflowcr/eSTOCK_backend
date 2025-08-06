package database

import "time"

type User struct {
	ID                string     `gorm:"column:id;primaryKey" json:"id"`
	Email             string     `gorm:"column:email;unique" json:"email"`
	FirstName         string     `gorm:"column:first_name" json:"first_name"`
	LastName          string     `gorm:"column:last_name" json:"last_name"`
	ProfileImageURL   *string    `gorm:"column:profile_image_url" json:"profile_image_url"`
	Password          *string    `gorm:"column:password" json:"-"`
	Role              string     `gorm:"column:role" json:"role"`
	IsActive          bool       `gorm:"column:is_active" json:"is_active"`
	AuthProvider      string     `gorm:"column:auth_provider" json:"auth_provider"`
	ResetToken        *string    `gorm:"column:reset_token" json:"-"`
	ResetTokenExpires *time.Time `gorm:"column:reset_token_expires" json:"-"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
