package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterGamificationRoutes(router *gin.RouterGroup, db *gorm.DB) {
	gamificationRepository := &repositories.GamificationRepository{DB: db}
	gamificationService := services.NewGamificationService(gamificationRepository)

	gamificationController := controllers.NewGamificationController(*gamificationService)

	route := router.Group("/gamification")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/stats", gamificationController.GamificationStats)
	}
}
