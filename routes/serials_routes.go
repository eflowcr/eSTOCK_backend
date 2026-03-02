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

var _ ports.SerialsRepository = (*repositories.SerialsRepository)(nil)

func RegisterSerialRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, serialService := wire.NewSerials(db)
	serialController := controllers.NewSerialsController(*serialService)

	route := router.Group("/serials")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/:id", serialController.GetSerialByID)
		route.GET("/by-sku/:sku", serialController.GetSerialsBySKU)
		route.POST("/", serialController.CreateSerial)
		route.PUT("/:id", serialController.UpdateSerial)
		route.DELETE("/:id", serialController.DeleteSerial)
	}
}
