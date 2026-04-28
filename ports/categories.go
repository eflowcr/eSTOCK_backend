package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// CategoriesRepository defines persistence operations for product categories (2-level tree).
type CategoriesRepository interface {
	Create(tenantID string, data *requests.CreateCategoryRequest) (*database.Category, *responses.InternalResponse)
	GetByID(id string) (*database.Category, *responses.InternalResponse)
	ListByTenant(tenantID string) ([]database.Category, *responses.InternalResponse)
	Update(id string, data *requests.UpdateCategoryRequest) (*database.Category, *responses.InternalResponse)
	SoftDelete(id string) *responses.InternalResponse
}
