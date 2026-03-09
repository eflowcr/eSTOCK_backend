package requests

type LocationTypeCreate struct {
	Code      string `json:"code" binding:"required" validate:"required,max=20"`
	Name      string `json:"name" binding:"required" validate:"required,max=100"`
	SortOrder int32  `json:"sort_order" validate:"gte=0"`
	IsActive  *bool  `json:"is_active"`
}

type LocationTypeUpdate struct {
	Code      string `json:"code" binding:"required" validate:"required,max=20"`
	Name      string `json:"name" binding:"required" validate:"required,max=100"`
	SortOrder int32  `json:"sort_order" validate:"gte=0"`
	IsActive  *bool  `json:"is_active"`
}
