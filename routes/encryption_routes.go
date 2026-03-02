package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
)

var _ ports.EncryptionRepository = (*repositories.EncryptionRepository)(nil)

func RegisterEncryptionRoutes(router *gin.RouterGroup, config configuration.Config) {
	_, encryptionService := wire.NewEncryption(config)
	encryptionController := controllers.NewEncryptionController(*encryptionService)

	route := router.Group("/encryption")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/encrypt/:data", encryptionController.EncryptData)
		route.GET("/decrypt/:data", encryptionController.DecryptData)
	}
}
