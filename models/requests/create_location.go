package requests

type Location struct {
	LocationCode string  `json:"location_code" binding:"required" validate:"required,max=50"`
	Description  *string `json:"description"`
	Zone         *string `json:"zone"`
	Type         string  `json:"type" binding:"required" validate:"required,max=50"`
}
