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
)

var _ ports.LocationTypesRepository = (*repositories.LocationTypesRepositorySQLC)(nil)

func RegisterLocationTypesRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, locationTypesService := wire.NewLocationTypes(pool)
	if locationTypesService == nil {
		return
	}
	locationTypesController := controllers.NewLocationTypesController(*locationTypesService)

	route := router.Group("/location-types")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		// Public (authenticated) list for dropdown - any user with locations read can see active types
		readLocations := tools.RequirePermission(rolesRepo, "locations", "read")
		route.GET("/", readLocations, locationTypesController.ListLocationTypes)

		// Admin: full list and CRUD (uses locations update permission)
		adminLocations := tools.RequirePermission(rolesRepo, "locations", "update")
		route.GET("/admin", adminLocations, locationTypesController.ListLocationTypesAdmin)
		route.GET("/:id", readLocations, locationTypesController.GetLocationTypeByID)
		route.POST("/", adminLocations, locationTypesController.CreateLocationType)
		route.PUT("/:id", adminLocations, locationTypesController.UpdateLocationType)
		route.DELETE("/:id", adminLocations, locationTypesController.DeleteLocationType)
	}
}
