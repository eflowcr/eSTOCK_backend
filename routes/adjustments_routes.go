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

var _ ports.AdjustmentsRepository = (*repositories.AdjustmentsRepository)(nil)

func RegisterAdjustmentsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, adjustmentsService := wire.NewAdjustments(db)
	adjustmentsController := controllers.NewAdjustmentsController(*adjustmentsService, config.JWTSecret)

	route := router.Group("/adjustments")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", adjustmentsController.GetAllAdjustments)
		route.GET("/:id", adjustmentsController.GetAdjustmentByID)
		route.GET("/:id/details", adjustmentsController.GetAdjustmentDetails)
		route.POST("/", adjustmentsController.CreateAdjustment)
		route.GET("/export", adjustmentsController.ExportAdjustmentsToExcel)
	}
}
