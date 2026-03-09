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

// RegisterPreferencesRoutes registers GET/PUT /api/user/preferences. Requires JWT auth.
func RegisterPreferencesRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config) {
	var prefsRepo ports.UserPreferencesRepository
	if pool != nil {
		prefsRepo = wire.NewUserPreferences(pool)
	}
	ctrl := controllers.NewPreferencesController(prefsRepo)
	route := router.Group("/user")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/preferences", ctrl.GetPreferences)
		route.PUT("/preferences", ctrl.UpdatePreferences)
	}
}
