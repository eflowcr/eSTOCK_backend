package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// ArticleSuppliersService — thin pass-through to the repository. Existence/type validation for
// supplier_id and article_sku is enforced by the FK constraints in migration 000022 (REFERENCES
// articles(sku) / clients(id)), so a bad reference surfaces as a DB error (handled as 500/400).
// Soft business rule "supplier must be type=supplier|both" is deferred (follow-up — would need
// ClientsRepository.GetByID exposed through ports; keep CRUD lean for now).
type ArticleSuppliersService struct {
	Repo ports.ArticleSuppliersRepository
}

func NewArticleSuppliersService(repo ports.ArticleSuppliersRepository) *ArticleSuppliersService {
	return &ArticleSuppliersService{Repo: repo}
}

func (s *ArticleSuppliersService) List(tenantID string, f ports.ArticleSupplierFilters) ([]database.ArticleSupplier, *responses.InternalResponse) {
	return s.Repo.List(tenantID, f)
}

func (s *ArticleSuppliersService) GetByID(id, tenantID string) (*database.ArticleSupplier, *responses.InternalResponse) {
	return s.Repo.GetByID(id, tenantID)
}

func (s *ArticleSuppliersService) Create(tenantID string, req *requests.CreateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse) {
	return s.Repo.Create(tenantID, req)
}

func (s *ArticleSuppliersService) Update(id, tenantID string, req *requests.UpdateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse) {
	return s.Repo.Update(id, tenantID, req)
}

func (s *ArticleSuppliersService) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return s.Repo.SoftDelete(id, tenantID)
}
