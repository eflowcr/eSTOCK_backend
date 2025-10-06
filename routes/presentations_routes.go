package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterPresentationsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	presentationsRepository := &repositories.PresentationsRepository{DB: db}
	presentationsService := services.NewPresentationsService(presentationsRepository)

	presentationsController := controllers.NewPresentationsController(*presentationsService)

	route := router.Group("/presentations")
	route.Use()
	{
		route.GET("/", presentationsController.GetAllPresentations)
		route.GET("/:id", presentationsController.GetPresentationByID)
		route.POST("/", presentationsController.CreatePresentation)
		route.PATCH("/:id", presentationsController.UpdatePresentation)
		route.DELETE("/:id", presentationsController.DeletePresentation)
	}
}
