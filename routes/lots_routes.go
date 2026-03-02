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

var _ ports.LotsRepository = (*repositories.LotsRepository)(nil)

func RegisterLotsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, lotsService := wire.NewLots(db)
	lotsController := controllers.NewLotsController(*lotsService)

	route := router.Group("/lots")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", lotsController.GetAllLots)
		route.GET("/:sku", lotsController.GetLotsBySKU)
		route.POST("/", lotsController.CreateLot)
		route.PUT("/:id", lotsController.UpdateLot)
		route.DELETE("/:id", lotsController.DeleteLot)
	}
}
