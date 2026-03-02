package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type EncryptionService struct {
	Repository ports.EncryptionRepository
}

func NewEncryptionService(repo ports.EncryptionRepository) *EncryptionService {
	return &EncryptionService{
		Repository: repo,
	}
}

func (s *EncryptionService) EncryptData(data string) (string, *responses.InternalResponse) {
	return s.Repository.Encrypt(data)
}

func (s *EncryptionService) DecryptData(encryptedData string) (string, *responses.InternalResponse) {
	return s.Repository.Decrypt(encryptedData)
}
