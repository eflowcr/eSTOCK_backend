package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLotsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	lotsRepository := &repositories.LotsRepository{DB: db}
	lotsService := services.NewLotsService(lotsRepository)

	lotsController := controllers.NewLotsController(*lotsService)

	route := router.Group("/lots")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", lotsController.GetAllLots)
		route.GET("/:sku", lotsController.GetLotsBySKU)
		route.POST("/", lotsController.CreateLot)
		route.PUT("/:id", lotsController.UpdateLot)
	}
}
