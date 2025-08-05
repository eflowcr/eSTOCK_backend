package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"golang.org/x/crypto/argon2"
)

// Encrypt encrypts data using Argon2 for key derivation and AES for encryption.
func Encrypt(plaintext string) (string, error) {
	password := configuration.Secret

	// Generate a salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Derive a key using Argon2
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create a GCM mode for encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Combine salt and ciphertext
	encryptedData := append(salt, ciphertext...)

	// Encode to base64 for easy storage/transmission
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// Decrypt decrypts data using Argon2 for key derivation and AES for decryption.
func Decrypt(encryptedData string) (string, error) {
	password := configuration.Secret
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	// Extract salt and ciphertext
	salt := data[:16]
	ciphertext := data[16:]

	// Derive the key using Argon2
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create a GCM mode for decryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ComparePasswords compares a hashed password with a plaintext password.
func ComparePasswords(hashedPassword, password string) bool {
	// Decrypt the hashed password
	hashedPassword, err := Decrypt(hashedPassword)
	if err != nil {
		return false
	}

	// Compare the passwords
	return hashedPassword == password
}
