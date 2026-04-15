package ports

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// AuthenticationRepository defines persistence operations for authentication.
type AuthenticationRepository interface {
	Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse)
	RequestPasswordReset(ctx context.Context, email string) *responses.InternalResponse
	ResetPassword(ctx context.Context, token, newPassword string) *responses.InternalResponse
}
