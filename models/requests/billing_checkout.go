package requests

// CheckoutRequest is the body for POST /api/billing/checkout.
// Plan must be one of: starter, pro, enterprise.
type CheckoutRequest struct {
	Plan string `json:"plan" binding:"required" validate:"required,oneof=starter pro enterprise"`
}
