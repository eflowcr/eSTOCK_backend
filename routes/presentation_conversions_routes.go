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

var _ ports.PresentationConversionsRepository = (*repositories.PresentationConversionsRepositorySQLC)(nil)

func RegisterPresentationConversionsRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, svc := wire.NewPresentationConversions(pool)
	if svc == nil {
		return
	}
	ctrl := controllers.NewPresentationConversionsController(*svc)

	route := router.Group("/presentation-conversions")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readArticles := tools.RequirePermission(rolesRepo, "articles", "read")
		adminArticles := tools.RequirePermission(rolesRepo, "articles", "update")

		route.GET("/", readArticles, ctrl.ListPresentationConversions)
		route.GET("/admin", adminArticles, ctrl.ListPresentationConversionsAdmin)
		route.GET("/:id", readArticles, ctrl.GetPresentationConversionByID)
		route.POST("/", adminArticles, ctrl.CreatePresentationConversion)
		route.PUT("/:id", adminArticles, ctrl.UpdatePresentationConversion)
		route.DELETE("/:id", adminArticles, ctrl.DeletePresentationConversion)
	}
}
