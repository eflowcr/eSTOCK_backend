package database

import "time"

// PresentationType represents a row in presentation_types (Unidad, Caja, Pallet, etc.).
type PresentationType struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	SortOrder int32     `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
