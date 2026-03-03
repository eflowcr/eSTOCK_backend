package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

var _ ports.PresentationsRepository = (*repositories.PresentationsRepository)(nil)
var _ ports.PresentationsRepository = (*repositories.PresentationsRepositorySQLC)(nil)

func RegisterPresentationsRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) {
	_, presentationsService := wire.NewPresentations(db, pool)
	presentationsController := controllers.NewPresentationsController(*presentationsService)

	route := router.Group("/presentations")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", presentationsController.GetAllPresentations)
		route.GET("/:id", presentationsController.GetPresentationByID)
		route.POST("/", presentationsController.CreatePresentation)
		route.PATCH("/:id", presentationsController.UpdatePresentation)
		route.DELETE("/:id", presentationsController.DeletePresentation)
	}
}
