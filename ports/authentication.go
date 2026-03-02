package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// AuthenticationRepository defines persistence operations for authentication.
type AuthenticationRepository interface {
	Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse)
}
