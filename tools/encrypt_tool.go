package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

// Encrypt encrypts data using Argon2 for key derivation and AES for encryption.
// password is the secret used for key derivation (e.g. JWT_SECRET).
func Encrypt(plaintext string, password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encryptedData := append(salt, ciphertext...)
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// Decrypt decrypts data using Argon2 for key derivation and AES for decryption.
func Decrypt(encryptedData string, password string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	salt := data[:16]
	ciphertext := data[16:]

	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ComparePasswords compares a hashed (encrypted) password with a plaintext password.
// secret is the encryption password (e.g. JWT_SECRET).
func ComparePasswords(hashedPassword, plainPassword, secret string) bool {
	decrypted, err := Decrypt(hashedPassword, secret)
	if err != nil {
		return false
	}
	return decrypted == plainPassword
}
