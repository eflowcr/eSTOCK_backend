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

var _ ports.PresentationTypesRepository = (*repositories.PresentationTypesRepositorySQLC)(nil)

func RegisterPresentationTypesRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, presentationTypesService := wire.NewPresentationTypes(pool)
	if presentationTypesService == nil {
		return
	}
	ctrl := controllers.NewPresentationTypesController(*presentationTypesService)

	route := router.Group("/presentation-types")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readArticles := tools.RequirePermission(rolesRepo, "articles", "read")
		adminArticles := tools.RequirePermission(rolesRepo, "articles", "update")

		route.GET("/", readArticles, ctrl.ListPresentationTypes)
		route.GET("/admin", adminArticles, ctrl.ListPresentationTypesAdmin)
		route.GET("/:id", readArticles, ctrl.GetPresentationTypeByID)
		route.POST("/", adminArticles, ctrl.CreatePresentationType)
		route.PUT("/:id", adminArticles, ctrl.UpdatePresentationType)
		route.DELETE("/:id", adminArticles, ctrl.DeletePresentationType)
	}
}
