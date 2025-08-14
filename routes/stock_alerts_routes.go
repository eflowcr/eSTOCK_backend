package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterStockAlertsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	stockAlertsRepository := &repositories.StockAlertsRepository{DB: db}
	stockAlertsService := services.NewStockAlertsService(stockAlertsRepository)

	stockAlertsController := controllers.NewStockAlertsController(*stockAlertsService)

	route := router.Group("/stock-alerts")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/:resolved", stockAlertsController.GetAllStockAlerts)
		route.GET("/analyze", stockAlertsController.Analyze)
		route.GET("/lot-expiration", stockAlertsController.LotExpiration)
		route.PATCH("/:id/resolve", stockAlertsController.ResolveAlert)
	}
}
