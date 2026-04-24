package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterCategoriesRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, categoriesService := wire.NewCategories(pool)
	categoriesController := controllers.NewCategoriesController(*categoriesService, config.TenantID)

	route := router.Group("/categories")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "categories", "read")
		write := tools.RequirePermission(rolesRepo, "categories", "write")

		route.GET("/", read, categoriesController.List)
		route.GET("/tree", read, categoriesController.GetTree)
		route.GET("/:id", read, categoriesController.GetByID)
		route.POST("/", write, categoriesController.Create)
		route.PATCH("/:id", write, categoriesController.Update)
		route.DELETE("/:id", write, categoriesController.SoftDelete)
	}
}
