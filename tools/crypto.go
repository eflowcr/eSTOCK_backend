package tools

import (
	"crypto/rand"
	"fmt"
)

// GenerateSecureToken returns n random bytes encoded as hex.
// Use for password reset tokens, API keys, etc.
func GenerateSecureToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
