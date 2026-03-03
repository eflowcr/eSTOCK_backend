package controllers

import (
	"encoding/json"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type RolesController struct {
	Repo ports.RolesRepository
}

func NewRolesController(repo ports.RolesRepository) *RolesController {
	return &RolesController{Repo: repo}
}

// ListRoles handles GET /api/roles. Returns all roles (admin only).
func (c *RolesController) ListRoles(ctx *gin.Context) {
	if c.Repo == nil {
		tools.ResponseInternal(ctx, "ListRoles", "Roles no disponibles", "list_roles")
		return
	}
	list, err := c.Repo.List(ctx.Request.Context())
	if err != nil {
		tools.ResponseInternal(ctx, "ListRoles", "Error al listar roles", "list_roles")
		return
	}
	tools.ResponseOK(ctx, "ListRoles", "Roles obtenidos", "list_roles", list, false, "")
}

// GetRoleByID handles GET /api/roles/:id.
func (c *RolesController) GetRoleByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetRoleByID", "get_role", "ID de rol inválido")
	if !ok {
		return
	}
	if c.Repo == nil {
		tools.ResponseInternal(ctx, "GetRoleByID", "Roles no disponibles", "get_role")
		return
	}
	role, err := c.Repo.GetByID(ctx.Request.Context(), id)
	if err != nil {
		tools.ResponseNotFound(ctx, "GetRoleByID", "Rol no encontrado", "get_role")
		return
	}
	tools.ResponseOK(ctx, "GetRoleByID", "Rol obtenido", "get_role", role, false, "")
}

// UpdateRolePermissionsRequest is the body for PUT /api/roles/:id.
type UpdateRolePermissionsRequest struct {
	Permissions json.RawMessage `json:"permissions"`
}

// UpdateRolePermissions handles PUT /api/roles/:id. Body: { "permissions": { "articles": { "create": true, "read": true }, ... } }.
func (c *RolesController) UpdateRolePermissions(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateRolePermissions", "update_role", "ID de rol inválido")
	if !ok {
		return
	}
	var req UpdateRolePermissionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateRolePermissions", "Cuerpo inválido; se espera { \"permissions\": { ... } }", "update_role")
		return
	}
	if len(req.Permissions) == 0 {
		tools.ResponseBadRequest(ctx, "UpdateRolePermissions", "permissions no puede estar vacío", "update_role")
		return
	}
	if c.Repo == nil {
		tools.ResponseInternal(ctx, "UpdateRolePermissions", "Roles no disponibles", "update_role")
		return
	}
	err := c.Repo.UpdatePermissions(ctx.Request.Context(), id, req.Permissions)
	if err != nil {
		tools.ResponseInternal(ctx, "UpdateRolePermissions", "Error al actualizar permisos", "update_role")
		return
	}
	role, _ := c.Repo.GetByID(ctx.Request.Context(), id)
	tools.ResponseOK(ctx, "UpdateRolePermissions", "Permisos actualizados", "update_role", role, false, "")
}
