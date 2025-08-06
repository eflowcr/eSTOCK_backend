package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type EncryptionService struct {
	Repository *repositories.EncryptionRepository
}

func NewEncryptionService(repo *repositories.EncryptionRepository) *EncryptionService {
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
