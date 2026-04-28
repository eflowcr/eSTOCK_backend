package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type UserService struct {
	Repository ports.UsersRepository
}

func NewUserService(repo ports.UsersRepository) *UserService {
	return &UserService{
		Repository: repo,
	}
}

func (s *UserService) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	return s.Repository.GetAllUsers()
}

func (s *UserService) GetUserByID(id string) (*database.User, *responses.InternalResponse) {
	return s.Repository.GetUserByID(id)
}

func (s *UserService) CreateUser(tenantID string, user *requests.User) *responses.InternalResponse {
	return s.Repository.CreateUser(tenantID, user)
}

func (s *UserService) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateUser(id, data)
}

func (s *UserService) DeleteUser(id string) *responses.InternalResponse {
	return s.Repository.DeleteUser(id)
}

func (s *UserService) ImportUsersFromExcel(tenantID string, fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportUsersFromExcel(tenantID, fileBytes)
}

func (s *UserService) ExportUsersToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportUsersToExcel()
}

func (s *UserService) UpdateUserPassword(id string, newPassword string) *responses.InternalResponse {
	return s.Repository.UpdateUserPassword(id, newPassword)
}

func (s *UserService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}
