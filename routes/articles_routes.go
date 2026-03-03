package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

var _ ports.ArticlesRepository = (*repositories.ArticlesRepository)(nil)
var _ ports.ArticlesRepository = (*repositories.ArticlesRepositorySQLC)(nil)

func RegisterArticlesRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, auditSvc *services.AuditService) {
	_, articlesService := wire.NewArticles(db, pool)
	articlesController := controllers.NewArticlesController(*articlesService, auditSvc)

	route := router.Group("/articles")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		// Legacy list endpoint (no generic table params); kept for compatibility.
		route.GET("/", articlesController.GetAllArticles)
		// Generic table list + CSV export (pagination, sort, filters, search).
		if pool != nil {
			cfg := tools.ArticlesTableConfig()
			route.GET("/table", tools.GenericListHandler(pool, cfg))
			route.GET("/table/export", tools.GenericExportHandler(pool, cfg, "articles.csv"))
		}
		route.GET("/:id", articlesController.GetArticleByID)
		route.GET("/sku/:sku", articlesController.GetBySku)
		route.POST("/", articlesController.CreateArticle)
		route.PUT("/:id", articlesController.UpdateArticle)
		route.POST("/import", articlesController.ImportArticlesFromExcel)
		route.GET("/export", articlesController.ExportArticlesToExcel)
		route.DELETE("/:id", articlesController.DeleteArticle)
	}
}
