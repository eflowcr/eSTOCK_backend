package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const encryptPassword = "super-secret-password"

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := "my-plain-password"
	encrypted, err := Encrypt(plaintext, encryptPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := Decrypt(encrypted, encryptPassword)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_ProducesUniqueValues(t *testing.T) {
	// Random salt means same input should produce different ciphertexts
	enc1, _ := Encrypt("same", encryptPassword)
	enc2, _ := Encrypt("same", encryptPassword)
	assert.NotEqual(t, enc1, enc2)
}

func TestDecrypt_WrongPassword(t *testing.T) {
	encrypted, err := Encrypt("secret", encryptPassword)
	require.NoError(t, err)

	_, err = Decrypt(encrypted, "wrong-password")
	assert.Error(t, err)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt("not-base64!!!", encryptPassword)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	// base64 of 20 zero bytes: 16-byte salt + 4 bytes ciphertext, which is less than
	// the GCM nonce size (12), so the code returns "ciphertext too short".
	short20 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAA" // base64(20 zero bytes)
	_, err := Decrypt(short20, encryptPassword)
	assert.Error(t, err)
}

func TestComparePasswords_Match(t *testing.T) {
	plain := "myPassword123"
	hashed, err := Encrypt(plain, encryptPassword)
	require.NoError(t, err)

	assert.True(t, ComparePasswords(hashed, plain, encryptPassword))
}

func TestComparePasswords_Mismatch(t *testing.T) {
	hashed, _ := Encrypt("correct", encryptPassword)
	assert.False(t, ComparePasswords(hashed, "wrong", encryptPassword))
}

func TestComparePasswords_BadHash(t *testing.T) {
	assert.False(t, ComparePasswords("invalid-hash", "anything", encryptPassword))
}
