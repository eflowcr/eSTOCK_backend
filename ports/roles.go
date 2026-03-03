package ports

import (
	"context"
	"encoding/json"
	"time"
)

// RolesRepository provides role data for RBAC and role management API.
type RolesRepository interface {
	GetRolePermissions(ctx context.Context, roleID string) ([]byte, error)
	List(ctx context.Context) ([]RoleEntry, error)
	GetByID(ctx context.Context, roleID string) (*RoleEntry, error)
	UpdatePermissions(ctx context.Context, roleID string, permissions json.RawMessage) error
}

// RoleEntry is a single role for API responses (list, get, update).
type RoleEntry struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Permissions json.RawMessage `json:"permissions"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
