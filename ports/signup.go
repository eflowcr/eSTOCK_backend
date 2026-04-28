package ports

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// SignupRepository defines persistence operations for self-service tenant signup.
type SignupRepository interface {
	// InitiateSignup validates uniqueness, stores a pending signup token, and sends a
	// verification email. Returns nil on success (202 Accepted path).
	// originURL is the request's Origin header value (may be empty); when it
	// matches the ALLOWED_ORIGINS allowlist the verification link is built from it,
	// otherwise the configured AppURL (or localhost dev fallback) is used.
	InitiateSignup(ctx context.Context, req requests.SignupRequest, originURL string) *responses.InternalResponse

	// VerifySignup atomically creates the tenant, admin user, and demo seed record,
	// then returns a JWT for immediate login.
	VerifySignup(ctx context.Context, token string) (*responses.SignupVerifiedResponse, *responses.InternalResponse)
}
