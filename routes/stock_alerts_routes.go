package routes

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var _ ports.StockAlertsRepository = (*repositories.StockAlertsRepository)(nil)

func RegisterStockAlertsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, redisClient *goredis.Client) {
	_, stockAlertsService := wire.NewStockAlerts(db, redisClient)
	stockAlertsController := controllers.NewStockAlertsController(*stockAlertsService)

	// 5 requests per minute per IP on the analyze endpoint (expensive: full DB scan + transaction).
	analyzeRateLimiter := tools.NewIPRateLimiter(rate.Every(time.Minute/5), 5)

	route := router.Group("/stock-alerts")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/:resolved", stockAlertsController.GetAllStockAlerts)
		route.GET("/analyze", analyzeRateLimiter, stockAlertsController.Analyze)
		route.GET("/lot-expiration", stockAlertsController.LotExpiration)
		route.PATCH("/:id/resolve", stockAlertsController.ResolveAlert)
		route.GET("/export", stockAlertsController.ExportAlertsToExcel)
	}
}
