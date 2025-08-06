package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterArticlesRoutes(router *gin.RouterGroup, db *gorm.DB) {
	articlesRepository := &repositories.ArticlesRepository{DB: db}
	articlesService := services.NewArticlesService(articlesRepository)

	articlesController := controllers.NewArticlesController(*articlesService)

	route := router.Group("/articles")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", articlesController.GetAllArticles)
		route.GET("/:id", articlesController.GetArticleByID)
		route.GET("/sku/:sku", articlesController.GetBySku)
		route.POST("/", articlesController.CreateArticle)
		route.PUT("/:id", articlesController.UpdateArticle)
		route.POST("/import", articlesController.ImportArticlesFromExcel)
		route.GET("/export", articlesController.ExportArticlesToExcel)
		route.DELETE("/:id", articlesController.DeleteArticle)
	}
}
