package requests

// PresentationConversionCreate is the request body for creating a presentation conversion.
type PresentationConversionCreate struct {
	FromPresentationTypeID string   `json:"from_presentation_type_id" binding:"required"`
	ToPresentationTypeID  string   `json:"to_presentation_type_id" binding:"required"`
	ConversionFactor      float64  `json:"conversion_factor" binding:"required" validate:"required,gt=0"`
	IsActive              *bool    `json:"is_active"`
}

// PresentationConversionUpdate is the request body for updating a presentation conversion.
type PresentationConversionUpdate struct {
	ConversionFactor float64 `json:"conversion_factor" binding:"required" validate:"required,gt=0"`
	IsActive         *bool   `json:"is_active"`
}
