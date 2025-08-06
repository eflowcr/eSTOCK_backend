package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
)

type EncryptionRepository struct {
}

func (e *EncryptionRepository) Encrypt(data string) (string, *responses.InternalResponse) {
	encryptedData, err := tools.Encrypt(data)
	if err != nil {
		return "", &responses.InternalResponse{
			Error:   err,
			Message: "Failed to encrypt data",
			Handled: false,
		}
	}

	return encryptedData, nil
}

func (e *EncryptionRepository) Decrypt(encryptedData string) (string, *responses.InternalResponse) {
	decryptedData, err := tools.Decrypt(encryptedData)
	if err != nil {
		return "", &responses.InternalResponse{
			Error:   err,
			Message: "Failed to decrypt data",
			Handled: false,
		}
	}

	return decryptedData, nil
}
