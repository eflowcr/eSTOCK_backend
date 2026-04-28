package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ArticlesRepository defines persistence operations for articles.
//
// S3.5 W1 — every HTTP-facing operation now requires a tenantID parameter
// (HR-S3-W5 C2 fix). The legacy non-tenant methods are retained ONLY for internal
// use cases that genuinely span tenants (FK resolution from inventory rows,
// stock-alerts cron, dashboards). They MUST NOT be reached from HTTP controllers.
//
// Implemented by *repositories.ArticlesRepositorySQLC (Postgres) and
// *repositories.ArticlesRepository (GORM fallback for sqlserver).
type ArticlesRepository interface {
	// ── tenant-scoped (HTTP-facing) ──────────────────────────────────────────
	GetAllArticlesForTenant(tenantID string) ([]database.Article, *responses.InternalResponse)
	GetArticleByIDForTenant(id, tenantID string) (*database.Article, *responses.InternalResponse)
	GetBySkuForTenant(sku, tenantID string) (*database.Article, *responses.InternalResponse)
	CreateArticleForTenant(tenantID string, data *requests.Article) *responses.InternalResponse
	UpdateArticleForTenant(id, tenantID string, data *requests.Article) (*database.Article, *responses.InternalResponse)
	DeleteArticleForTenant(id, tenantID string) *responses.InternalResponse
	ImportArticlesFromExcelForTenant(tenantID string, fileBytes []byte) ([]string, []string, []*responses.InternalResponse)
	ImportArticlesFromJSONForTenant(tenantID string, rows []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse)
	ValidateImportRowsForTenant(tenantID string, rows []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse)
	ExportArticlesToExcelForTenant(tenantID string) ([]byte, *responses.InternalResponse)
	GenerateImportTemplateForTenant(tenantID, language string) ([]byte, *responses.InternalResponse)

	// ── internal (no tenant filter) ──────────────────────────────────────────
	// These remain for stock-alerts cron, FK resolution from inventory.sku/lots.sku,
	// and other system-level reads that legitimately span tenants.
	GetAllArticles() ([]database.Article, *responses.InternalResponse)
	GetArticleByID(id string) (*database.Article, *responses.InternalResponse)
	GetBySku(sku string) (*database.Article, *responses.InternalResponse)
	GetLotsBySKU(sku string) ([]database.Lot, error)
	GetSerialsBySKU(sku string) ([]database.Serial, error)
}
