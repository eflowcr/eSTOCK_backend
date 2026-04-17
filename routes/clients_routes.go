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

func RegisterClientsRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, clientsService := wire.NewClients(pool)
	clientsController := controllers.NewClientsController(*clientsService, config.TenantID)

	route := router.Group("/clients")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "clients", "read")
		write := tools.RequirePermission(rolesRepo, "clients", "write")

		route.GET("/", read, clientsController.List)
		route.GET("/:id", read, clientsController.GetByID)
		route.POST("/", write, clientsController.Create)
		route.PATCH("/:id", write, clientsController.Update)
		route.DELETE("/:id", write, clientsController.SoftDelete)
	}
}
