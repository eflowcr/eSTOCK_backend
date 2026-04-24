package requests

type CreateClientRequest struct {
	Type    string  `json:"type" binding:"required" validate:"required,oneof=supplier customer both"`
	Code    string  `json:"code" binding:"required" validate:"required,max=50"`
	Name    string  `json:"name" binding:"required" validate:"required,max=200"`
	Email   *string `json:"email" validate:"omitempty,email,max=150"`
	Phone   *string `json:"phone" validate:"omitempty,max=50"`
	Address *string `json:"address" validate:"omitempty"`
	TaxID   *string `json:"tax_id" validate:"omitempty,max=50"`
	Notes   *string `json:"notes" validate:"omitempty"`
}
