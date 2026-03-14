package requests

// AdjustmentReasonCodeCreate is the request body for creating an adjustment reason code.
type AdjustmentReasonCodeCreate struct {
	Code         string `json:"code" binding:"required" validate:"required,max=80"`
	Name         string `json:"name" binding:"required" validate:"required,max=255"`
	Direction    string `json:"direction" binding:"required" validate:"required,oneof=inbound outbound"`
	DisplayOrder int32  `json:"display_order" validate:"gte=0"`
	IsActive     *bool  `json:"is_active"`
}

// AdjustmentReasonCodeUpdate is the request body for updating an adjustment reason code.
type AdjustmentReasonCodeUpdate struct {
	Code         string `json:"code" binding:"required" validate:"required,max=80"`
	Name         string `json:"name" binding:"required" validate:"required,max=255"`
	Direction    string `json:"direction" binding:"required" validate:"required,oneof=inbound outbound"`
	DisplayOrder int32  `json:"display_order" validate:"gte=0"`
	IsActive     *bool  `json:"is_active"`
}
