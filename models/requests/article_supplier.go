package requests

// CreateArticleSupplierRequest is the body for POST /api/article-suppliers/.
// All numeric fields default to nil (omitted) so callers can patch only what they need.
type CreateArticleSupplierRequest struct {
	ArticleSKU   string   `json:"article_sku" binding:"required" validate:"required,max=64"`
	SupplierID   string   `json:"supplier_id" binding:"required" validate:"required,max=64"`
	IsPreferred  *bool    `json:"is_preferred" validate:"omitempty"`
	LeadTimeDays *int     `json:"lead_time_days" validate:"omitempty,min=0,max=3650"`
	UnitCost     *float64 `json:"unit_cost" validate:"omitempty,min=0"`
	SupplierSKU  *string  `json:"supplier_sku" validate:"omitempty,max=128"`
	Notes        *string  `json:"notes" validate:"omitempty,max=1000"`
}

// UpdateArticleSupplierRequest is the body for PATCH /api/article-suppliers/:id.
// article_sku and supplier_id are NOT patchable (they define the relationship — recreate to change).
type UpdateArticleSupplierRequest struct {
	IsPreferred  *bool    `json:"is_preferred" validate:"omitempty"`
	LeadTimeDays *int     `json:"lead_time_days" validate:"omitempty,min=0,max=3650"`
	UnitCost     *float64 `json:"unit_cost" validate:"omitempty,min=0"`
	SupplierSKU  *string  `json:"supplier_sku" validate:"omitempty,max=128"`
	Notes        *string  `json:"notes" validate:"omitempty,max=1000"`
}
