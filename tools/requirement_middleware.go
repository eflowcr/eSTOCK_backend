package tools

import (
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
)

// permSourceHeader exposes whether RequirePermission authorized via the JWT-embedded
// permissions claim or the DB fallback. Useful for ops dashboards (cache-hit ratio) and
// debugging "why is this 403" without enabling verbose logging. Header values are
// constants so frontends/log shippers can grep on a stable string.
const (
	permSourceHeader = "X-Perm-Source"
	permSourceJWT    = "jwt"
	permSourceDB     = "db"
)

// RequirePermission returns Gin middleware that enforces RBAC. As of S3.8 the permissions
// blob is read from the JWT-embedded claim first (set by JWTAuthMiddleware on the gin
// context) and only falls back to a DB lookup when the claim is absent — the typical
// reason being a token issued before S3.8 that hasn't expired yet. Denies with 401 if
// no auth/role/tenant, 403 if permission missing.
//
// Backwards compat: pre-S3.8 tokens lack the permissions claim, so they keep working via
// the DB lookup until they expire. Once everyone has re-logged-in (≤ token TTL), the DB
// lookup path becomes dead code we could trim — but keep it for now as the safe fallback
// for any future code path that bypasses the new login (e.g. machine-to-machine tokens).
//
// store may be nil for tests; the middleware then short-circuits to Next(). Must run
// after JWTAuthMiddleware (so ContextKeyUserID, ContextKeyRole and ContextKeyTenantID
// are set).
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

		// S3.8 — JWT-cache fast path. PermissionsFromContext returns nil if the claim
		// was absent (legacy/pre-S3.8 token); in that case fall through to the DB lookup
		// so existing tokens keep working until they expire. Permission *changes* on the
		// server are NOT reflected until the user re-logs in (or until the token expires
		// and gets re-issued from the new role state) — same trade-off as the cached DB
		// path which already had a 2-min TTL. Acceptable because the caching wrapper had
		// the same property.
		if jwtPerms := PermissionsFromContext(c); jwtPerms != nil {
			c.Header(permSourceHeader, permSourceJWT)
			if !HasPermission(jwtPerms, resource, action) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permiso insuficiente"})
				return
			}
			c.Next()
			return
		}

		// Fallback: pre-S3.8 token (no permissions claim) — load from store.
		c.Header(permSourceHeader, permSourceDB)
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
