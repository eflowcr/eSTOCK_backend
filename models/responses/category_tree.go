package responses

type CategoryTreeNode struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	ParentID *string            `json:"parent_id,omitempty"`
	IsActive bool               `json:"is_active"`
	Children []CategoryTreeNode `json:"children"`
}
