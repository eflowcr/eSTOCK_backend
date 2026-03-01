package requests

type Login struct {
	Email    string `json:"email" binding:"required" validate:"required,email,max=255"`
	Password string `json:"password" binding:"required" validate:"required,min=1"`
}
