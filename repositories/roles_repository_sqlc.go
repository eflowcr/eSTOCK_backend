package repositories

import (
	"context"
	"encoding/json"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// RolesRepositorySQLC implements ports.RolesRepository using sqlc.
type RolesRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewRolesRepositorySQLC returns a roles repository backed by sqlc.
func NewRolesRepositorySQLC(queries *sqlc.Queries) *RolesRepositorySQLC {
	return &RolesRepositorySQLC{queries: queries}
}

var _ ports.RolesRepository = (*RolesRepositorySQLC)(nil)

func (r *RolesRepositorySQLC) GetRolePermissions(ctx context.Context, roleID string) ([]byte, error) {
	raw, err := r.queries.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (r *RolesRepositorySQLC) List(ctx context.Context) ([]ports.RoleEntry, error) {
	list, err := r.queries.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ports.RoleEntry, len(list))
	for i, row := range list {
		out[i] = sqlcRoleToEntry(row)
	}
	return out, nil
}

func (r *RolesRepositorySQLC) GetByID(ctx context.Context, roleID string) (*ports.RoleEntry, error) {
	row, err := r.queries.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	e := sqlcRoleToEntry(row)
	return &e, nil
}

func (r *RolesRepositorySQLC) UpdatePermissions(ctx context.Context, roleID string, permissions json.RawMessage) error {
	_, err := r.queries.UpdateRolePermissions(ctx, sqlc.UpdateRolePermissionsParams{
		ID:          roleID,
		Permissions: permissions,
	})
	return err
}

func sqlcRoleToEntry(row sqlc.Role) ports.RoleEntry {
	desc := ""
	if row.Description.Valid {
		desc = row.Description.String
	}
	return ports.RoleEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: desc,
		Permissions: row.Permissions,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
