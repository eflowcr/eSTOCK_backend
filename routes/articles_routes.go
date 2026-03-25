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

func RegisterArticlesRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, auditSvc *services.AuditService, rolesRepo ports.RolesRepository) {
	_, articlesService := wire.NewArticles(db, pool)
	articlesController := controllers.NewArticlesController(*articlesService, auditSvc)

	route := router.Group("/articles")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "articles", "read")
		create := tools.RequirePermission(rolesRepo, "articles", "create")
		update := tools.RequirePermission(rolesRepo, "articles", "update")
		delete := tools.RequirePermission(rolesRepo, "articles", "delete")

		route.GET("/", read, articlesController.GetAllArticles)
		if pool != nil {
			cfg := tools.ArticlesTableConfig()
			route.GET("/table", read, tools.GenericListHandler(pool, cfg))
			route.GET("/table/export", read, tools.GenericExportHandler(pool, cfg, "articles.csv"))
		}
		route.GET("/import/template", read, articlesController.DownloadImportTemplate)
		route.GET("/:id", read, articlesController.GetArticleByID)
		route.GET("/sku/:sku", read, articlesController.GetBySku)
		route.POST("/", create, articlesController.CreateArticle)
		route.PUT("/:id", update, articlesController.UpdateArticle)
		route.POST("/import", create, articlesController.ImportArticlesFromExcel)
		route.GET("/export", read, articlesController.ExportArticlesToExcel)
		route.DELETE("/:id", delete, articlesController.DeleteArticle)
	}
}
