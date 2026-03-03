package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

var _ ports.LocationsRepository = (*repositories.LocationsRepository)(nil)
var _ ports.LocationsRepository = (*repositories.LocationsRepositorySQLC)(nil)

func RegisterLocationRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) {
	_, locationService := wire.NewLocations(db, pool)
	locationController := controllers.NewLocationsController(*locationService)

	route := router.Group("/locations")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", locationController.GetAllLocations)
		route.GET("/:id", locationController.GetLocationByID)
		route.POST("/", locationController.CreateLocation)
		route.PUT("/:id", locationController.UpdateLocation)
		route.DELETE("/:id", locationController.DeleteLocation)
		route.POST("/import", locationController.ImportLocationsFromExcel)
		route.GET("/export", locationController.ExportLocationsToExcel)
	}
}
