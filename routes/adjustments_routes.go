package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAdjustmentsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	adjustmentsRepository := &repositories.AdjustmentsRepository{DB: db}
	adjustmentsService := services.NewAdjustmentsService(adjustmentsRepository)

	adjustmentsController := controllers.NewAdjustmentsController(*adjustmentsService)

	route := router.Group("/adjustments")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", adjustmentsController.GetAllAdjustments)
		route.GET("/:id", adjustmentsController.GetAdjustmentByID)
		route.GET("/:id/details", adjustmentsController.GetAdjustmentDetails)
		route.POST("/", adjustmentsController.CreateAdjustment)
		route.GET("/export", adjustmentsController.ExportAdjustmentsToExcel)
	}
}
