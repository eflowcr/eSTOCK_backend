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

var _ ports.SerialsRepository = (*repositories.SerialsRepository)(nil)
var _ ports.SerialsRepository = (*repositories.SerialsRepositorySQLC)(nil)

func RegisterSerialRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	_, serialService := wire.NewSerials(db, pool)
	// S3.5 W2-A: pass tenantID from configuration so all CRUD is tenant-scoped.
	serialController := controllers.NewSerialsController(*serialService, config.TenantID)

	route := router.Group("/serials")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "serials", "read")
		create := tools.RequirePermission(rolesRepo, "serials", "create")
		update := tools.RequirePermission(rolesRepo, "serials", "update")
		delete := tools.RequirePermission(rolesRepo, "serials", "delete")

		route.GET("/:id", read, serialController.GetSerialByID)
		route.GET("/by-sku/:sku", read, serialController.GetSerialsBySKU)
		route.POST("/", create, serialController.CreateSerial)
		route.PUT("/:id", update, serialController.UpdateSerial)
		route.DELETE("/:id", delete, serialController.DeleteSerial)
	}
}
