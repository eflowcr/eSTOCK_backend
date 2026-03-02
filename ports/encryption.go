package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// EncryptionRepository defines operations for encryption/decryption.
type EncryptionRepository interface {
	Encrypt(data string) (string, *responses.InternalResponse)
	Decrypt(encryptedData string) (string, *responses.InternalResponse)
}
