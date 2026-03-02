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

var _ ports.GamificationRepository = (*repositories.GamificationRepository)(nil)

func RegisterGamificationRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, gamificationService := wire.NewGamification(db)
	gamificationController := controllers.NewGamificationController(*gamificationService, config.JWTSecret)

	route := router.Group("/gamification")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/stats", gamificationController.GamificationStats)
		route.GET("/badges", gamificationController.Badges)
		route.GET("/all-badges", gamificationController.GetAllBadges)
		route.POST("/complete-tasks", gamificationController.CompleteTasks)
		route.GET("/operator-stats", gamificationController.GetAllUserStats)
	}
}
