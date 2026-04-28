package requests

// SignupRequest is the body for POST /api/signup.
// All fields are required. TenantSlug must match ^[a-z0-9-]{3,32}$.
//
// SeedDemoData (S3.7 companion / B23) is OPTIONAL — pointer so the controller
// can distinguish "not sent" (nil → default true for backwards compat) from
// "explicitly false" (opt-out). The frontend's signup form ships this as a
// checkbox defaulting to OFF; users who want demo data tick it on, signup
// payloads from older clients that omit the field continue to seed.
type SignupRequest struct {
	Email        string `json:"email"         validate:"required,email"`
	CompanyName  string `json:"company_name"  validate:"required,min=2,max=120"`
	TenantSlug   string `json:"tenant_slug"   validate:"required,min=3,max=32"`
	AdminName    string `json:"admin_name"    validate:"required,min=2,max=80"`
	AdminPassword string `json:"admin_password" validate:"required,min=8"`
	SeedDemoData *bool  `json:"seed_demo_data,omitempty"`
}

// SignupVerifyRequest is the body for POST /api/signup/verify.
type SignupVerifyRequest struct {
	Token string `json:"token" validate:"required"`
}
