package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthenticationRoutes(router *gin.RouterGroup, db *gorm.DB) {
	authenticationRepository := &repositories.AuthenticationRepository{DB: db}
	authenticationService := services.NewAuthenticationService(authenticationRepository)

	authenticationController := controllers.NewAuthenticationController(*authenticationService)

	route := router.Group("/auth")
	{
		route.POST("/login", authenticationController.Login)
	}
}
