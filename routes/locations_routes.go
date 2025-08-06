package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLocationRoutes(router *gin.RouterGroup, db *gorm.DB) {
	locationRepository := &repositories.LocationsRepository{DB: db}
	locationService := services.NewLocationsService(locationRepository)

	locationController := controllers.NewLocationsController(*locationService)

	route := router.Group("/locations")
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
