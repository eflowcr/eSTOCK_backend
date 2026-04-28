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

func RegisterStockSettingsRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, stockSettingsService := wire.NewStockSettings(pool)
	stockSettingsController := controllers.NewStockSettingsController(*stockSettingsService, config.TenantID)

	route := router.Group("/settings/stock")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "settings", "read")
		write := tools.RequirePermission(rolesRepo, "settings", "write")

		route.GET("", read, stockSettingsController.Get)
		route.PATCH("", write, stockSettingsController.Update)
	}
}
