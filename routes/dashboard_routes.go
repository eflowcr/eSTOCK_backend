package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterDashboardRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	dashboardRepository := &repositories.DashboardRepository{DB: db}
	dashboardService := services.NewDashboardService(dashboardRepository)

	dashboardController := controllers.NewDashboardController(*dashboardService)

	route := router.Group("/dashboard")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/stats", dashboardController.GetDashboardStats)
	}
}
