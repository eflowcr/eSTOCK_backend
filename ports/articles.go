package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ArticlesRepository defines persistence operations for articles.
// Implemented by *repositories.ArticlesRepository (GORM).
type ArticlesRepository interface {
	GetAllArticles() ([]database.Article, *responses.InternalResponse)
	GetArticleByID(id string) (*database.Article, *responses.InternalResponse)
	GetBySku(sku string) (*database.Article, *responses.InternalResponse)
	CreateArticle(data *requests.Article) *responses.InternalResponse
	UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse)
	GetLotsBySKU(sku string) ([]database.Lot, error)
	GetSerialsBySKU(sku string) ([]database.Serial, error)
	ImportArticlesFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse)
	ExportArticlesToExcel() ([]byte, *responses.InternalResponse)
	DeleteArticle(id string) *responses.InternalResponse
}
