package requests

// SignupRequest is the body for POST /api/signup.
// All fields are required. TenantSlug must match ^[a-z0-9-]{3,32}$.
type SignupRequest struct {
	Email        string `json:"email"         validate:"required,email"`
	CompanyName  string `json:"company_name"  validate:"required,min=2,max=120"`
	TenantSlug   string `json:"tenant_slug"   validate:"required,min=3,max=32"`
	AdminName    string `json:"admin_name"    validate:"required,min=2,max=80"`
	AdminPassword string `json:"admin_password" validate:"required,min=8"`
}

// SignupVerifyRequest is the body for POST /api/signup/verify.
type SignupVerifyRequest struct {
	Token string `json:"token" validate:"required"`
}
