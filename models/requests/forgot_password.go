package requests

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required,min=32"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}
