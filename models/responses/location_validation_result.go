package responses

import "github.com/eflowcr/eSTOCK_backend/models/requests"

// LocationValidationStatus describes the result of validating one import row against the DB.
type LocationValidationStatus string

const (
	LocationStatusNew       LocationValidationStatus = "new"
	LocationStatusExists    LocationValidationStatus = "exists"
	LocationStatusSimilar   LocationValidationStatus = "similar"
	LocationStatusError     LocationValidationStatus = "error"
	LocationStatusDuplicate LocationValidationStatus = "duplicate"
)

// LocationValidationMatch is a compact representation of an existing DB location.
type LocationValidationMatch struct {
	ID           string `json:"id"`
	LocationCode string `json:"location_code"`
	Description  string `json:"description"`
	Zone         string `json:"zone"`
	Type         string `json:"type"`
	IsActive     bool   `json:"is_active"`
}

// LocationValidationResult is the per-row output of the validate endpoint.
type LocationValidationResult struct {
	RowIndex          int                      `json:"row_index"`
	Status            LocationValidationStatus `json:"status"`
	Row               requests.LocationImportRow `json:"row"`
	FieldErrors       map[string]string        `json:"field_errors,omitempty"`
	ExistingLocation  *LocationValidationMatch  `json:"existing_location,omitempty"`
	SimilarLocations  []LocationValidationMatch `json:"similar_locations,omitempty"`
}
