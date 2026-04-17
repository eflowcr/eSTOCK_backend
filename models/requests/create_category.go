package requests

type CreateCategoryRequest struct {
	Name     string  `json:"name" binding:"required" validate:"required,max=150"`
	ParentID *string `json:"parent_id" validate:"omitempty"`
}
