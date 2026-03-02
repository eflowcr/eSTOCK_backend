package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.PresentationsRepository = (*repositories.PresentationsRepository)(nil)

func RegisterPresentationsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, presentationsService := wire.NewPresentations(db)
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
