package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

func RegisterEncryptionRoutes(router *gin.RouterGroup, config configuration.Config) {
	encryptionRepository := &repositories.EncryptionRepository{JWTSecret: config.JWTSecret}
	encryptionService := services.NewEncryptionService(encryptionRepository)

	encryptionController := controllers.NewEncryptionController(*encryptionService)

	route := router.Group("/encryption")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/encrypt/:data", encryptionController.EncryptData)
		route.GET("/decrypt/:data", encryptionController.DecryptData)
	}
}
