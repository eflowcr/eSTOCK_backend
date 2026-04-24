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

var _ ports.DashboardRepository = (*repositories.DashboardRepository)(nil)

func RegisterDashboardRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	_, dashboardService := wire.NewDashboard(db)
	dashboardController := controllers.NewDashboardController(*dashboardService)

	route := router.Group("/dashboard")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "dashboard", "read")

		route.GET("/stats", read, dashboardController.GetDashboardStats)
		route.GET("/inventory-summary", read, dashboardController.GetInventorySummary)
		route.GET("/movements-monthly", read, dashboardController.GetMovementsMonthly)
		route.GET("/activity", read, dashboardController.GetRecentActivity)
	}
}
