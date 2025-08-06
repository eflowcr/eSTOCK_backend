package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type AuthenticationService struct {
	Repository *repositories.AuthenticationRepository
}

func NewAuthenticationService(repo *repositories.AuthenticationRepository) *AuthenticationService {
	return &AuthenticationService{
		Repository: repo,
	}
}

func (s *AuthenticationService) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	return s.Repository.Login(login)
}
