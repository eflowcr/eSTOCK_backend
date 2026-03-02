package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.AuthenticationRepository = (*repositories.AuthenticationRepository)(nil)

func RegisterAuthenticationRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, authenticationService := wire.NewAuthentication(db, config)
	authenticationController := controllers.NewAuthenticationController(*authenticationService)

	route := router.Group("/auth")
	{
		route.POST("/login", authenticationController.Login)
	}
}
