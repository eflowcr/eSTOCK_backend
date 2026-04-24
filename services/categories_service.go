package services

import (
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type CategoriesService struct {
	Repository ports.CategoriesRepository
}

func NewCategoriesService(repo ports.CategoriesRepository) *CategoriesService {
	return &CategoriesService{Repository: repo}
}

func (s *CategoriesService) Create(tenantID string, data *requests.CreateCategoryRequest) (*database.Category, *responses.InternalResponse) {
	// Validate parent exists if set
	if data.ParentID != nil {
		parent, resp := s.Repository.GetByID(*data.ParentID)
		if resp != nil {
			return nil, resp
		}
		if parent == nil {
			return nil, &responses.InternalResponse{
				Message:    "Categoría padre no encontrada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		// Enforce 2-level max: root categories have no parent, children have a root parent
		if parent.ParentID != nil {
			return nil, &responses.InternalResponse{
				Message:    "Solo se permiten 2 niveles de jerarquía en categorías",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
	}
	return s.Repository.Create(tenantID, data)
}

func (s *CategoriesService) GetByID(id string) (*database.Category, *responses.InternalResponse) {
	return s.Repository.GetByID(id)
}

func (s *CategoriesService) ListByTenant(tenantID string) ([]database.Category, *responses.InternalResponse) {
	return s.Repository.ListByTenant(tenantID)
}

// ListByTenantFiltered delegates to the SQL-filtered repository path (M8).
// Pass nil for any parameter to skip that filter.
func (s *CategoriesService) ListByTenantFiltered(tenantID string, isActive *bool, search *string, limit *int32, offset *int32) ([]database.Category, *responses.InternalResponse) {
	return s.Repository.ListByTenantFiltered(tenantID, isActive, search, limit, offset)
}

func (s *CategoriesService) GetTree(tenantID string) ([]responses.CategoryTreeNode, *responses.InternalResponse) {
	all, resp := s.Repository.ListByTenant(tenantID)
	if resp != nil {
		return nil, resp
	}

	// Build tree in-memory: two passes
	childrenMap := make(map[string][]responses.CategoryTreeNode)
	var roots []responses.CategoryTreeNode

	for _, cat := range all {
		node := responses.CategoryTreeNode{
			ID:       cat.ID,
			Name:     cat.Name,
			ParentID: cat.ParentID,
			IsActive: cat.IsActive,
			Children: []responses.CategoryTreeNode{},
		}
		if cat.ParentID == nil {
			roots = append(roots, node)
		} else {
			childrenMap[*cat.ParentID] = append(childrenMap[*cat.ParentID], node)
		}
	}

	// Attach children to roots
	for i, root := range roots {
		if children, ok := childrenMap[root.ID]; ok {
			roots[i].Children = children
		}
	}

	if roots == nil {
		roots = []responses.CategoryTreeNode{}
	}
	return roots, nil
}

func (s *CategoriesService) Update(id string, data *requests.UpdateCategoryRequest, tenantID string) (*database.Category, *responses.InternalResponse) {
	existing, resp := s.Repository.GetByID(id)
	if resp != nil {
		return nil, resp
	}

	// Validate parent exists and 2-level constraint
	if data.ParentID != nil {
		// Prevent self-reference
		if *data.ParentID == id {
			return nil, &responses.InternalResponse{
				Message:    "Una categoría no puede ser su propio padre",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}

		parent, pResp := s.Repository.GetByID(*data.ParentID)
		if pResp != nil {
			return nil, pResp
		}
		if parent == nil {
			return nil, &responses.InternalResponse{
				Message:    "Categoría padre no encontrada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		if parent.ParentID != nil {
			return nil, &responses.InternalResponse{
				Message:    "Solo se permiten 2 niveles de jerarquía en categorías",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}

		// Cycle check: the new parent must not be a child of the current node
		all, listResp := s.Repository.ListByTenant(tenantID)
		if listResp != nil {
			return nil, listResp
		}
		if wouldCreateCycle(all, id, *data.ParentID) {
			return nil, &responses.InternalResponse{
				Message:    "El cambio de padre crearía un ciclo en la jerarquía",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
	}

	_ = existing
	return s.Repository.Update(id, data)
}

func (s *CategoriesService) SoftDelete(id string) *responses.InternalResponse {
	return s.Repository.SoftDelete(id)
}

// wouldCreateCycle checks if setting nodeID's parent to newParentID would create a cycle.
func wouldCreateCycle(categories []database.Category, nodeID, newParentID string) bool {
	parentMap := make(map[string]*string, len(categories))
	for _, c := range categories {
		id := c.ID
		parentMap[id] = c.ParentID
	}

	current := &newParentID
	visited := make(map[string]bool)
	for current != nil {
		if *current == nodeID {
			return true
		}
		if visited[*current] {
			return false
		}
		visited[*current] = true
		current = parentMap[*current]
	}
	return false
}

// containsIgnoreCase reports whether s contains substr case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
