package responses

// SignupInitiatedResponse is returned from POST /api/signup (202 Accepted).
type SignupInitiatedResponse struct {
	Message string `json:"message"`
}

// SignupVerifiedResponse is returned from POST /api/signup/verify (201 Created).
// Contains a ready-to-use JWT for immediate login.
type SignupVerifiedResponse struct {
	Token    string `json:"token"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}
