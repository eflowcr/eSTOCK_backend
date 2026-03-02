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

var _ ports.StockAlertsRepository = (*repositories.StockAlertsRepository)(nil)

func RegisterStockAlertsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, stockAlertsService := wire.NewStockAlerts(db)
	stockAlertsController := controllers.NewStockAlertsController(*stockAlertsService)

	route := router.Group("/stock-alerts")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/:resolved", stockAlertsController.GetAllStockAlerts)
		route.GET("/analyze", stockAlertsController.Analyze)
		route.GET("/lot-expiration", stockAlertsController.LotExpiration)
		route.PATCH("/:id/resolve", stockAlertsController.ResolveAlert)
		route.GET("/export", stockAlertsController.ExportAlertsToExcel)
	}
}
