// MobileUsersController is the mobile-only Users admin facade hit at
// /api/mobile/users and /api/mobile/roles. It is ADMIN-gated at the route level
// (RequirePermission(rolesRepo, "users", read|update); roles with permissions.all
// qualify).
//
// Design rule (same as MobileController): handlers are thin adapters over the
// EXISTING UserService — no business logic is duplicated. In particular user
// creation reuses UserService.CreateUser so password hashing (tools.Encrypt /
// Argon2+AES with JWT_SECRET, NOT bcrypt), email-uniqueness, role-name resolution
// and tenant stamping are byte-identical to the web path. The web /api/users
// routes are NOT touched.
//
// Trimmed DTOs (MobileUserDto / MobileRoleDto) never leak the password hash or the
// tenant UUID.
package controllers

import (
	"strings"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type MobileUsersController struct {
	Users     *services.UserService
	RolesRepo ports.RolesRepository // for the form's role picker (GET /roles); may be nil in test mode
	Config    configuration.Config
}

func NewMobileUsersController(users *services.UserService, rolesRepo ports.RolesRepository, config configuration.Config) *MobileUsersController {
	return &MobileUsersController{
		Users:     users,
		RolesRepo: rolesRepo,
		Config:    config,
	}
}

// MobileCreateUserRequest is the create payload. password is required on create;
// role_id accepts a roles.id or a role name (Admin/Operator/Viewer) — the service
// resolves it. name is split into first/last for the underlying requests.User
// (the service derives the display name from first+last, falling back to email).
type MobileCreateUserRequest struct {
	Name     string `json:"name" validate:"required,max=200"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=6"`
	RoleID   string `json:"role_id" validate:"required,max=50"`
}

// MobileUpdateUserRequest is the PATCH payload. Both fields are optional pointers
// so the client can send is_active alone (activate/deactivate), role_id alone
// (reassign role), or both. Nothing else is mutable from mobile (name/email/
// password edits stay web-only for this scope).
type MobileUpdateUserRequest struct {
	IsActive *bool   `json:"is_active,omitempty"`
	RoleID   *string `json:"role_id,omitempty" validate:"omitempty,max=50"`
}

// ─── GET /api/mobile/users ────────────────────────────────────────────────────

func (c *MobileUsersController) ListUsers(ctx *gin.Context) {
	if c.Users == nil {
		tools.ResponseOK(ctx, "MobileListUsers", "Sin usuarios", "mobile_list_users", []responses.MobileUserDto{}, false, "")
		return
	}
	// Tenant scope: the mobile admin must only list users inside their own tenant
	// (same JWT-sourced contract as CreateUser) — prevents a cross-tenant user leak.
	tenantID := tools.ResolveTenantID(ctx, c.Config.TenantID)
	if tenantID == "" {
		tools.ResponseUnauthorized(ctx, "MobileListUsers", "tenant no identificado", "mobile_list_users")
		return
	}
	users, resp := c.Users.GetUsersByTenant(tenantID)
	if resp != nil {
		writeErrorResponse(ctx, "MobileListUsers", "mobile_list_users", resp)
		return
	}

	out := make([]responses.MobileUserDto, 0, len(users))
	for _, u := range users {
		roleName := ""
		if u.Role != nil {
			roleName = u.Role.Name
		}
		out = append(out, responses.MobileUserDto{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			RoleID:   u.RoleID,
			RoleName: roleName,
			IsActive: u.IsActive,
		})
	}
	tools.ResponseOK(ctx, "MobileListUsers", "Usuarios obtenidos", "mobile_list_users", out, false, "")
}

// ─── POST /api/mobile/users ───────────────────────────────────────────────────

func (c *MobileUsersController) CreateUser(ctx *gin.Context) {
	if c.Users == nil {
		tools.ResponseInternal(ctx, "MobileCreateUser", "Servicio no disponible", "mobile_create_user")
		return
	}

	var body MobileCreateUserRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "MobileCreateUser", "Cuerpo de solicitud inválido", "mobile_create_user")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "MobileCreateUser", "mobile_create_user", errs)
		return
	}

	// Tenant scope: prefer the JWT-sourced tenant on the context (set by
	// JWTAuthMiddleware), fall back to the configured single-tenant id. Same
	// contract the web CreateUser relies on via RequirePermission.
	tenantID := tools.ResolveTenantID(ctx, c.Config.TenantID)
	if tenantID == "" {
		tools.ResponseUnauthorized(ctx, "MobileCreateUser", "tenant no identificado", "mobile_create_user")
		return
	}

	// Split display name into first/last for the underlying requests.User. The
	// service re-derives the canonical Name from first+last, so a single-word
	// name (no space) becomes first=word, last="" → Name=word.
	first, last := splitDisplayName(body.Name)
	password := body.Password
	req := &requests.User{
		Email:     strings.TrimSpace(body.Email),
		FirstName: first,
		LastName:  last,
		Password:  &password,
		RoleID:    body.RoleID,
	}

	// Reuse the EXISTING service method → identical password hashing
	// (tools.Encrypt), email-uniqueness check and role resolution as web.
	if resp := c.Users.CreateUser(tenantID, req); resp != nil {
		writeErrorResponse(ctx, "MobileCreateUser", "mobile_create_user", resp)
		return
	}

	tools.ResponseCreated(ctx, "MobileCreateUser", "Usuario creado con éxito", "mobile_create_user", nil, false, "")
}

// ─── PATCH /api/mobile/users/:id ──────────────────────────────────────────────

func (c *MobileUsersController) UpdateUser(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "MobileUpdateUser", "mobile_update_user", "ID de usuario inválido")
	if !ok {
		return
	}
	if c.Users == nil {
		tools.ResponseInternal(ctx, "MobileUpdateUser", "Servicio no disponible", "mobile_update_user")
		return
	}

	var body MobileUpdateUserRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "MobileUpdateUser", "Cuerpo de solicitud inválido", "mobile_update_user")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "MobileUpdateUser", "mobile_update_user", errs)
		return
	}

	// Build the whitelist of mutable fields. Anything not present is left
	// untouched. password/id/created_at are stripped by the service regardless.
	data := map[string]interface{}{}
	if body.IsActive != nil {
		data["is_active"] = *body.IsActive
	}
	if body.RoleID != nil && strings.TrimSpace(*body.RoleID) != "" {
		data["role_id"] = strings.TrimSpace(*body.RoleID) // service resolves name→id
	}
	if len(data) == 0 {
		tools.ResponseBadRequest(ctx, "MobileUpdateUser", "Nada para actualizar (envía is_active y/o role_id)", "mobile_update_user")
		return
	}

	if resp := c.Users.UpdateUser(id, data); resp != nil {
		writeErrorResponse(ctx, "MobileUpdateUser", "mobile_update_user", resp)
		return
	}

	tools.ResponseOK(ctx, "MobileUpdateUser", "Usuario actualizado con éxito", "mobile_update_user", nil, false, "")
}

// ─── GET /api/mobile/roles ────────────────────────────────────────────────────

func (c *MobileUsersController) ListRoles(ctx *gin.Context) {
	if c.RolesRepo == nil {
		tools.ResponseOK(ctx, "MobileListRoles", "Sin roles", "mobile_list_roles", []responses.MobileRoleDto{}, false, "")
		return
	}
	roles, err := c.RolesRepo.List(ctx.Request.Context())
	if err != nil {
		tools.ResponseInternal(ctx, "MobileListRoles", "Error al listar roles", "mobile_list_roles")
		return
	}
	out := make([]responses.MobileRoleDto, 0, len(roles))
	for _, r := range roles {
		out = append(out, responses.MobileRoleDto{ID: r.ID, Name: r.Name})
	}
	tools.ResponseOK(ctx, "MobileListRoles", "Roles obtenidos", "mobile_list_roles", out, false, "")
}

// splitDisplayName splits a free-form display name into a (firstName, lastName)
// pair on the first space. A single token returns (token, ""). Leading/trailing
// space is trimmed. The underlying service re-derives the canonical Name from
// these, so the round-trip is lossless for the common "First Last" case.
func splitDisplayName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
