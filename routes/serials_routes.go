package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterSerialRoutes(router *gin.RouterGroup, db *gorm.DB) {
	serialRepository := &repositories.SerialsRepository{DB: db}
	serialService := services.NewSerialsService(serialRepository)

	serialController := controllers.NewSerialsController(*serialService)

	route := router.Group("/serials")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/:id", serialController.GetSerialByID)
		route.GET("/by-sku/:sku", serialController.GetSerialsBySKU)
		route.POST("/", serialController.CreateSerial)
		route.PUT("/:id", serialController.UpdateSerial)
		route.DELETE("/:id", serialController.DeleteSerial)
	}
}
