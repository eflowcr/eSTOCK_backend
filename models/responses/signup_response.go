package responses

import "encoding/json"

// SignupInitiatedResponse is returned from POST /api/signup (202 Accepted).
type SignupInitiatedResponse struct {
	Message string `json:"message"`
}

// SignupVerifiedResponse is returned from POST /api/signup/verify (201 Created).
// Contains a ready-to-use JWT for immediate login.
//
// S3.5.6 B22: Role + Permissions are populated by the service layer (mirroring
// AuthenticationService.Login) so the auto-login post-verify produces a fully
// hydrated frontend session. Without these fields the menu collapses to
// Dashboard only because isAdmin()/hasPermission() see undefined.
//
// RoleID is the internal role identifier the repo persists; the service uses it
// to look up the role name + permissions and is stripped before serialization
// (json:"-") since the frontend only needs the resolved name.
type SignupVerifiedResponse struct {
	Token       string          `json:"token"`
	TenantID    string          `json:"tenant_id"`
	Email       string          `json:"email"`
	Name        string          `json:"name"`
	Role        string          `json:"role,omitempty"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
	RoleID      string          `json:"-"`
}
