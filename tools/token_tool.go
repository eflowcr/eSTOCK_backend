package tools

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserId   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT signed with the given secret.
//
// TODO(ARCH — S3.5 blocker): JWT Claims has no tenant_id field. Controllers resolve tenant from
// Config.TenantID (env var), not from the token. In a multi-tenant SaaS, a user from Tenant A
// can use their JWT against Tenant B's billing endpoint because the controller uses the static
// env-var TenantID. Adding tenant_id to Claims + validating it in RequirePermission middleware
// is required before opening signups to multiple real tenants on the same pod.
// Tracked as S3.5-jwt-tenant-claim. See: feedback_estock_articles_no_tenant_isolation.md
func GenerateToken(secret string, userId, userName, email, role string) (string, error) {
	claims := Claims{
		UserId:   userId,
		UserName: userName,
		Email:    email,
		Role:     role,
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
