package responses

type LoginResponse struct {
	Name     string `json:"name"`
	LastName string `json:"last_name"`
	Email    string `json:"email"`
	Token    string `json:"token"`
	Role     string `json:"role"`
}
