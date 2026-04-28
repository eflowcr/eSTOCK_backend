package services

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// SignupService orchestrates the self-service tenant signup flow.
type SignupService struct {
	Repository      ports.SignupRepository
	RolesRepository ports.RolesRepository // optional: when set, enriches verify response with role name + permissions
}

// NewSignupService constructs SignupService.
//
// rolesRepo is optional; when nil the verify response carries only the JWT
// (legacy behavior). When provided, the service mirrors
// AuthenticationService.Login by attaching the role name + permissions blob to
// the verify response so the frontend's auto-login produces a fully hydrated
// session (see B22 / S3.5.6).
func NewSignupService(repo ports.SignupRepository, rolesRepo ports.RolesRepository) *SignupService {
	return &SignupService{
		Repository:      repo,
		RolesRepository: rolesRepo,
	}
}

// InitiateSignup validates the request and starts the email verification flow.
func (s *SignupService) InitiateSignup(ctx context.Context, req requests.SignupRequest) *responses.InternalResponse {
	return s.Repository.InitiateSignup(ctx, req)
}

// VerifySignup completes the signup: creates tenant, admin user, demo data.
//
// S3.5.6 B22: when RolesRepository is wired, the response is enriched with the
// role name (e.g. "Admin") and the permissions JSON blob the frontend uses for
// isAdmin() / hasPermission() gating. Without enrichment the menu collapses to
// Dashboard only after the auto-login post-verify because permissions stays
// undefined until a manual logout+login.
//
// Enrichment failures are non-fatal: the JWT is already valid, so the user can
// always recover with a logout+login. We log a warn and return the unenriched
// response rather than fail the whole signup.
func (s *SignupService) VerifySignup(ctx context.Context, token string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	resp, errResp := s.Repository.VerifySignup(ctx, token)
	if errResp != nil || resp == nil {
		return resp, errResp
	}

	if s.RolesRepository != nil && resp.RoleID != "" {
		if perms, err := s.RolesRepository.GetRolePermissions(ctx, resp.RoleID); err == nil && len(perms) > 0 {
			resp.Permissions = perms
		}
		if roleEntry, err := s.RolesRepository.GetByID(ctx, resp.RoleID); err == nil && roleEntry != nil {
			resp.Role = roleEntry.Name
		}
	}

	return resp, nil
}
