package database

import "time"

// Role is the GORM model for the roles table (RBAC). id = nanoid; name is the stable identifier.
type Role struct {
	ID          string    `gorm:"column:id;primaryKey" json:"id"`
	Name        string    `gorm:"column:name" json:"name"`
	Description *string   `gorm:"column:description" json:"description"`
	IsActive    bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}
