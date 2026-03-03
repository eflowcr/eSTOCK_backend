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
