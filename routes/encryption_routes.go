package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
)

func RegisterEncryptionRoutes(router *gin.RouterGroup) {
	encryptionRepository := &repositories.EncryptionRepository{}
	encryptionService := services.NewEncryptionService(encryptionRepository)

	encryptionController := controllers.NewEncryptionController(*encryptionService)

	route := router.Group("/encryption")
	{
		route.GET("/encrypt/:data", encryptionController.EncryptData)
		route.GET("/decrypt/:data", encryptionController.DecryptData)
	}
}
