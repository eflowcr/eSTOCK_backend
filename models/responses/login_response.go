package responses

import "encoding/json"

// LoginResponse is returned on successful login. Permissions are loaded from the
// role store so the client can enforce route and UI visibility without extra requests.
type LoginResponse struct {
	Name        string          `json:"name"`
	LastName    string          `json:"last_name"`
	Email       string          `json:"email"`
	Token       string          `json:"token"`
	Role        string          `json:"role"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
}
