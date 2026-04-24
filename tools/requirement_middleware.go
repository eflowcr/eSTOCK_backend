package tools

import (
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
)

// RequirePermission returns Gin middleware that enforces RBAC: role from JWT context,
// permissions loaded server-side from store (never from token). Denies with 401 if no
// auth/role, 403 if permission missing. For enterprise: permission changes take effect
// after cache TTL without re-login.
// Must run after JWTAuthMiddleware (so ContextKeyUserID and ContextKeyRole are set).
func RequirePermission(store ports.RolesRepository, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.Next()
			return
		}

		roleVal, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no autorizado"})
			return
		}
		role, ok := roleVal.(string)
		if !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "rol no válido"})
			return
		}

		// S3.5 W3 — reject tokens lacking the tenant_id claim. This forces a re-login for any
		// JWT issued before the W3 deploy (claim was absent → defaults to ""). Without this gate
		// a stale token would silently fall through to controllers that read TenantIDFromContext
		// and get an empty string, which they would either reject (good) or treat as a wildcard
		// (very bad). Reject here so the failure is consistent and actionable: 401 → re-login.
		tenantVal, _ := c.Get(ContextKeyTenantID)
		tenantID, _ := tenantVal.(string)
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token sin tenant — vuelve a iniciar sesión",
			})
			return
		}

		perms, err := store.GetRolePermissions(c.Request.Context(), role)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron verificar permisos"})
			return
		}

		if !HasPermission(perms, resource, action) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permiso insuficiente"})
			return
		}

		c.Next()
	}
}
