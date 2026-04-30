package services

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AuthenticationService struct {
	Repository   ports.AuthenticationRepository
	RolesRepository ports.RolesRepository // optional: used to attach permissions to login response
}

// NewAuthenticationService builds the auth service. If rolesRepo is non-nil, login
// response will include permissions for the user's role.
func NewAuthenticationService(repo ports.AuthenticationRepository, rolesRepo ports.RolesRepository) *AuthenticationService {
	return &AuthenticationService{
		Repository:      repo,
		RolesRepository: rolesRepo,
	}
}

func (s *AuthenticationService) RequestPasswordReset(ctx context.Context, email string, originURL string) *responses.InternalResponse {
	return s.Repository.RequestPasswordReset(ctx, email, originURL)
}

func (s *AuthenticationService) ResetPassword(ctx context.Context, token, newPassword string) *responses.InternalResponse {
	return s.Repository.ResetPassword(ctx, token, newPassword)
}

func (s *AuthenticationService) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	resp, errResp := s.Repository.Login(login)
	if errResp != nil || resp == nil {
		return resp, errResp
	}
	roleID := resp.Role
	if s.RolesRepository != nil && roleID != "" {
		perms, err := s.RolesRepository.GetRolePermissions(context.Background(), roleID)
		if err == nil && len(perms) > 0 {
			resp.Permissions = perms
		}
		// Return role name (not id) for frontend; code column was removed, name is the identifier.
		roleEntry, err := s.RolesRepository.GetByID(context.Background(), roleID)
		if err == nil && roleEntry != nil {
			resp.Role = roleEntry.Name
		}
	}
	return resp, nil
}
