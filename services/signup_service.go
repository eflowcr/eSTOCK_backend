package services

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// SignupService orchestrates the self-service tenant signup flow.
type SignupService struct {
	Repository ports.SignupRepository
}

// NewSignupService constructs SignupService.
func NewSignupService(repo ports.SignupRepository) *SignupService {
	return &SignupService{Repository: repo}
}

// InitiateSignup validates the request and starts the email verification flow.
func (s *SignupService) InitiateSignup(ctx context.Context, req requests.SignupRequest) *responses.InternalResponse {
	return s.Repository.InitiateSignup(ctx, req)
}

// VerifySignup completes the signup: creates tenant, admin user, demo data.
func (s *SignupService) VerifySignup(ctx context.Context, token string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	return s.Repository.VerifySignup(ctx, token)
}
