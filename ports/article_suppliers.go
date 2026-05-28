package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ArticleSupplierFilters bundles optional ?article_sku=, ?supplier_id=, ?is_preferred= query filters.
// Nil = "do not filter on this field".
type ArticleSupplierFilters struct {
	ArticleSKU  *string
	SupplierID  *string
	IsPreferred *bool
}

// ArticleSuppliersRepository defines persistence operations for the article↔supplier link table.
// All operations are tenant-scoped (cross-tenant access is impossible — every method receives tenantID).
type ArticleSuppliersRepository interface {
	List(tenantID string, filters ArticleSupplierFilters) ([]database.ArticleSupplier, *responses.InternalResponse)
	GetByID(id, tenantID string) (*database.ArticleSupplier, *responses.InternalResponse)
	Create(tenantID string, req *requests.CreateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse)
	Update(id, tenantID string, req *requests.UpdateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse)
	SoftDelete(id, tenantID string) *responses.InternalResponse
}
