package database

import "time"

type Client struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Type      string     `json:"type"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Email     *string    `json:"email,omitempty"`
	Phone     *string    `json:"phone,omitempty"`
	Address   *string    `json:"address,omitempty"`
	TaxID     *string    `json:"tax_id,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedBy *string    `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
