package database

import "time"

type User struct {
	ID                string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID          string     `gorm:"column:tenant_id;type:uuid;not null" json:"-"` // S3.5 W5.5 (HR C2): per-user tenant scope; stamped into JWT at login
	Name              string     `gorm:"column:name;not null" json:"name"`
	Email             string     `gorm:"column:email" json:"email"`
	FirstName         string     `gorm:"column:first_name" json:"first_name"`
	LastName          string     `gorm:"column:last_name" json:"last_name"`
	ProfileImageURL   *string    `gorm:"column:profile_image_url" json:"profile_image_url"`
	Password          *string    `gorm:"column:password" json:"-"`
	RoleID            string     `gorm:"column:role_id;not null" json:"role_id"`
	Role              *Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	IsActive          bool       `gorm:"column:is_active" json:"is_active"`
	EmailVerified     bool       `gorm:"column:email_verified" json:"email_verified"`
	EmailVerifiedAt   *time.Time `gorm:"column:email_verified_at" json:"email_verified_at"`
	UpdatedBy         *string    `gorm:"column:updated_by" json:"-"`
	DeletedAt         *time.Time `gorm:"column:deleted_at" json:"-"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
