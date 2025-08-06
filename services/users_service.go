package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type UserService struct {
	Repository *repositories.UsersRepository
}

func NewUserService(repo *repositories.UsersRepository) *UserService {
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

func (s *UserService) CreateUser(user *requests.User) *responses.InternalResponse {
	return s.Repository.CreateUser(user)
}

func (s *UserService) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateUser(id, data)
}

func (s *UserService) DeleteUser(id string) *responses.InternalResponse {
	return s.Repository.DeleteUser(id)
}
