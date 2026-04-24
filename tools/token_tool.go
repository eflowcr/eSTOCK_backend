package tools

import (
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
type Claims struct {
	UserId   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT signed with the given secret. tenantID MUST be non-empty —
// callers (login, signup verify, refresh) must source it explicitly:
//   - login: Config.TenantID (single-tenant pilot) or user.TenantID once users have a
//     tenant column (future wave).
//   - signup verify: the freshly created tenant's UUID.
func GenerateToken(secret string, userId, userName, email, role, tenantID string) (string, error) {
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
