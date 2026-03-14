package database

import "time"

// PresentationConversion represents a conversion rule: 1 unit of From = ConversionFactor units of To.
type PresentationConversion struct {
	ID                      string    `json:"id"`
	FromPresentationTypeID  string    `json:"from_presentation_type_id"`
	ToPresentationTypeID    string    `json:"to_presentation_type_id"`
	ConversionFactor        float64   `json:"conversion_factor"`
	IsActive                bool      `json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}
