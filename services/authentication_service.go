package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AuthenticationService struct {
	Repository ports.AuthenticationRepository
}

func NewAuthenticationService(repo ports.AuthenticationRepository) *AuthenticationService {
	return &AuthenticationService{
		Repository: repo,
	}
}

func (s *AuthenticationService) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	return s.Repository.Login(login)
}
