package tools

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Context keys for values set by JWTAuthMiddleware (e.g. audit, RBAC, multi-tenant isolation).
const (
	ContextKeyUserID      = "user_id"
	ContextKeyRole        = "role"
	ContextKeyTenantID    = "tenant_id"   // S3.5 W3 — tenant isolation per request
	ContextKeyPermissions = "permissions" // S3.8 — JWT-embedded permissions blob (optional, may be nil)
)

// JWTAuthMiddleware returns a Gin middleware that validates JWT and sets user_id and role on context.
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secretKey := []byte(secret)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token no proporcionado"})
			c.Abort()
			return
		}

		tokenString := strings.Split(authHeader, "Bearer ")
		if len(tokenString) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de token inválido"})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(tokenString[1], &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok {
			c.Set(ContextKeyUserID, claims.UserId)
			c.Set(ContextKeyRole, claims.Role)
			// S3.5 W3 — surface tenant_id so controllers can scope per-request without
			// touching Config.TenantID env var. Empty value is intentionally still set so
			// RequirePermission can detect and reject pre-W3 tokens.
			c.Set(ContextKeyTenantID, claims.TenantID)
			// S3.8 — surface signed permissions blob so RequirePermission can authorize
			// without a per-request DB lookup. Pre-S3.8 tokens carry no claim → leave
			// the key unset so RequirePermission falls back to the DB lookup (backwards
			// compat). We deliberately do NOT set the key with an empty value because
			// PermissionsFromContext treats "absent" and "empty" identically and the
			// fallback path keys off c.Get exists.
			if len(claims.Permissions) > 0 {
				c.Set(ContextKeyPermissions, claims.Permissions)
			}
			// Also set "email" for legacy callers (BillingController used to read it).
			if claims.Email != "" {
				c.Set("email", claims.Email)
			}
		}
		c.Next()
	}
}
