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

func RegisterLocationRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	_, locationService := wire.NewLocations(db, pool)
	locationController := controllers.NewLocationsController(*locationService)

	route := router.Group("/locations")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "locations", "read")
		create := tools.RequirePermission(rolesRepo, "locations", "create")
		update := tools.RequirePermission(rolesRepo, "locations", "update")
		delete := tools.RequirePermission(rolesRepo, "locations", "delete")

		route.GET("/", read, locationController.GetAllLocations)
		if pool != nil {
			cfg := tools.LocationsTableConfig()
			route.GET("/table", read, tools.GenericListHandler(pool, cfg))
			route.GET("/table/export", read, tools.GenericExportHandler(pool, cfg, "locations.csv"))
		}
		route.GET("/:id", read, locationController.GetLocationByID)
		route.POST("/", create, locationController.CreateLocation)
		route.PUT("/:id", update, locationController.UpdateLocation)
		route.DELETE("/:id", delete, locationController.DeleteLocation)
		route.GET("/import/template", read, locationController.DownloadImportTemplate)
		route.POST("/import", create, locationController.ImportLocationsFromExcel)
		route.GET("/export", read, locationController.ExportLocationsToExcel)
	}
}
