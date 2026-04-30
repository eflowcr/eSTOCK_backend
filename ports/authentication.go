package ports

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// AuthenticationRepository defines persistence operations for authentication.
type AuthenticationRepository interface {
	Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse)
	// RequestPasswordReset generates and emails a password-reset link.
	// originURL is the request's Origin header value (may be empty); when it
	// matches the ALLOWED_ORIGINS allowlist the link is built from it,
	// otherwise the configured AppURL (or localhost dev fallback) is used.
	RequestPasswordReset(ctx context.Context, email string, originURL string) *responses.InternalResponse
	ResetPassword(ctx context.Context, token, newPassword string) *responses.InternalResponse
}
