package requests

// PresentationTypeCreate is the request body for creating a presentation type.
type PresentationTypeCreate struct {
	Code      string `json:"code" binding:"required" validate:"required,max=20"`
	Name      string `json:"name" binding:"required" validate:"required,max=100"`
	SortOrder int32  `json:"sort_order" validate:"gte=0"`
	IsActive  *bool  `json:"is_active"`
}

// PresentationTypeUpdate is the request body for updating a presentation type.
type PresentationTypeUpdate struct {
	Code      string `json:"code" binding:"required" validate:"required,max=20"`
	Name      string `json:"name" binding:"required" validate:"required,max=100"`
	SortOrder int32  `json:"sort_order" validate:"gte=0"`
	IsActive  *bool  `json:"is_active"`
}
