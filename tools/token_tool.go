package tools

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserId   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userId, userName, email, role string) (string, error) {
	claims := Claims{
		UserId:   userId,
		UserName: userName,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2400)),
			Issuer:    "EWIKI-API",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// secret := Encrypt(config.Secret)

	tokenString, err := token.SignedString([]byte(configuration.Secret))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserId(tokenString string) (string, error) {
	// Elimina el prefijo "Bearer " si está presente
	if len(tokenString) > 7 && strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = tokenString[7:]
	}

	// Usa el secreto tal cual, sin cifrar
	secret := []byte(configuration.Secret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Asegurar que el método de firma es el esperado
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return "", err
	}

	// Verifica que el token sea válido
	if !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	return claims.UserId, nil
}

func GetUserName(tokenString string) (string, error) {
	tokenString = tokenString[7:]

	secret, _ := Encrypt(configuration.Secret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims := token.Claims.(*Claims)

	return claims.UserName, nil
}

func GetEmail(tokenString string) (string, error) {
	tokenString = tokenString[7:]

	secret, _ := Encrypt(configuration.Secret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims := token.Claims.(*Claims)

	return claims.Email, nil
}

func GetRole(tokenString string) (string, error) {
	tokenString = tokenString[7:]

	secret, _ := Encrypt(configuration.Secret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims := token.Claims.(*Claims)

	return claims.Role, nil
}
