package requests

type Location struct {
	LocationCode string  `json:"location_code" binding:"required"`
	Description  *string `json:"description"`
	Zone         *string `json:"zone"`
	Type         string  `json:"type" binding:"required"`
}
