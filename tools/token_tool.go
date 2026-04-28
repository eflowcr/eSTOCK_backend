package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Claims is the JWT payload for authenticated users.
//
// S3.5 W3: Added TenantID to enable per-request tenant isolation. Controllers must read
// the tenant from gin.Context (TenantIDFromContext) instead of Config.TenantID env var so
// a single pod can serve multiple tenants safely. Tokens issued before W3 do not carry
// this claim and will be rejected by RequirePermission (forces re-login). Acceptable since
// v0.2.1 is a hotfix deploy with very few active users.
//
// S3.8: Added Permissions to embed the role's permissions JSONB inside the signed JWT,
// allowing RequirePermission to authorize requests without a per-request DB lookup. Tokens
// issued before S3.8 lack this claim — RequirePermission falls back to the DB lookup so
// pre-S3.8 tokens keep working until they expire (backwards compat). Permissions are
// snapshot at login/verify time; role updates take effect at next token issuance (or sooner
// once the cache TTL elapses if the fallback path is exercised, e.g. permissions claim absent).
type Claims struct {
	UserId      string          `json:"user_id"`
	UserName    string          `json:"user_name"`
	Email       string          `json:"email"`
	Role        string          `json:"role"`
	TenantID    string          `json:"tenant_id"`
	Permissions json.RawMessage `json:"permissions,omitempty"` // S3.8 — optional; missing → DB fallback
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT signed with the given secret. tenantID MUST be non-empty —
// callers (login, signup verify, refresh) must source it explicitly:
//   - login: Config.TenantID (single-tenant pilot) or user.TenantID once users have a
//     tenant column (future wave).
//   - signup verify: the freshly created tenant's UUID.
//
// permissions is OPTIONAL (S3.8): when non-nil + non-empty it is embedded in the signed
// claims so RequirePermission middleware can authorize without a DB roundtrip. Pass nil
// (or empty) to mint a legacy-shape token; RequirePermission will then fall back to the
// DB lookup. Callers that have the role's permissions in scope (login, signup verify)
// should pass them; system/test paths that don't can pass nil.
func GenerateToken(secret string, userId, userName, email, role, tenantID string, permissions json.RawMessage) (string, error) {
	claims := Claims{
		UserId:   userId,
		UserName: userName,
		Email:    email,
		Role:     role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			// TODO(M5 — S3.5): 2400h = 100-day expiry. JWTs issued to self-signup trial users should
			// be short-lived (e.g. 24h) with refresh. A cancelled subscriber retains access for 99 days.
			// Requires a token revocation mechanism (blocklist or short TTL + refresh token). S3.5 scope.
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2400)),
			Issuer:    "EWIKI-API",
		},
	}
	// Only embed when caller actually supplied a non-empty blob. Empty slice would still
	// serialize (json:"omitempty" skips zero-length json.RawMessage), but be explicit.
	if len(permissions) > 0 {
		claims.Permissions = permissions
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// TenantIDFromContext returns the tenant_id claim that JWTAuthMiddleware placed on the
// gin.Context. Returns "" if absent — callers (controllers) MUST treat empty as a failure
// because RequirePermission already rejects tokens lacking the claim. Returning empty here
// is a defense-in-depth: an endpoint not behind RequirePermission still gets a safe zero value.
func TenantIDFromContext(c *gin.Context) string {
	v, ok := c.Get(ContextKeyTenantID)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// PermissionsFromContext returns the permissions JSON blob that JWTAuthMiddleware
// placed on the gin.Context after decoding the JWT. Returns nil if the claim was
// absent (legacy/pre-S3.8 token) or wasn't a json.RawMessage. RequirePermission uses
// this to decide between JWT-cache authorization and the DB-lookup fallback.
func PermissionsFromContext(c *gin.Context) json.RawMessage {
	v, ok := c.Get(ContextKeyPermissions)
	if !ok {
		return nil
	}
	raw, ok := v.(json.RawMessage)
	if !ok {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	return raw
}

// ResolveTenantID returns the tenant for this request: JWT claim first, fallback only
// if the claim is missing. Returns "" iff there is no tenant available — callers MUST
// then return 401 to avoid leaking another tenant's data via the env var fallback.
//
// S3.5 W5.5 (HR-S3.5 C1): every tenant-scoped controller uses this helper to source
// the tenant from the JWT instead of the env-injected Config.TenantID. The fallback
// covers system/cron/admin paths that bypass JWTAuthMiddleware (e.g. test rigs that
// pre-construct a controller with a default tenant); HTTP requests behind
// JWTAuthMiddleware always get the JWT claim.
func ResolveTenantID(c *gin.Context, fallback string) string {
	if t := TenantIDFromContext(c); t != "" {
		return t
	}
	return fallback
}

func GetUserId(secret string, tokenString string) (string, error) {
	if len(tokenString) > 7 && strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = tokenString[7:]
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", errors.New("invalid token claims")
	}
	return claims.UserId, nil
}

func GetUserName(secret string, tokenString string) (string, error) {
	tokenString = tokenString[7:]
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims := token.Claims.(*Claims)
	return claims.UserName, nil
}

func GetEmail(secret string, tokenString string) (string, error) {
	tokenString = tokenString[7:]
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims := token.Claims.(*Claims)
	return claims.Email, nil
}

// IsOperatorRole returns true when the role string corresponds to a warehouse
// operator. Comparison is case-insensitive and trim-tolerant. Used by mobile
// list handlers to force assigned_to_me=true so an operator never sees tasks
// assigned to another operator (W7 N2-1: cross-operator data leak fix).
func IsOperatorRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "operator")
}

func GetRole(secret string, tokenString string) (string, error) {
	tokenString = tokenString[7:]
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims := token.Claims.(*Claims)
	return claims.Role, nil
}
