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

func RegisterNotificationsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, notifSvc *services.NotificationsService) {
	if db == nil || notifSvc == nil {
		return
	}

	repo := &repositories.NotificationsRepository{DB: db}
	notifController := controllers.NewNotificationsController(repo, config.TenantID)

	route := router.Group("/notifications")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", notifController.List)
		route.GET("/count", notifController.CountUnread)
		route.PATCH("/mark-all-read", notifController.MarkAllRead)
		route.PATCH("/:id/read", notifController.MarkRead)
		route.GET("/preferences", notifController.GetPreferences)
		route.PUT("/preferences", notifController.UpsertPreferences)
	}
}
