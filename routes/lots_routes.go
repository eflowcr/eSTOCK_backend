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

var _ ports.LotsRepository = (*repositories.LotsRepository)(nil)
var _ ports.LotsRepository = (*repositories.LotsRepositorySQLC)(nil)

func RegisterLotsRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	_, lotsService := wire.NewLots(db, pool)
	lotsController := controllers.NewLotsController(*lotsService)

	route := router.Group("/lots")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "lots", "read")
		create := tools.RequirePermission(rolesRepo, "lots", "create")
		update := tools.RequirePermission(rolesRepo, "lots", "update")
		delete := tools.RequirePermission(rolesRepo, "lots", "delete")

		route.GET("/", read, lotsController.GetAllLots)
		if pool != nil {
			cfg := tools.LotsTableConfig()
			route.GET("/table", read, tools.GenericListHandler(pool, cfg))
			route.GET("/table/export", read, tools.GenericExportHandler(pool, cfg, "lots.csv"))
		}
		route.GET("/:sku", read, lotsController.GetLotsBySKU)
		route.POST("/", create, lotsController.CreateLot)
		route.PUT("/:id", update, lotsController.UpdateLot)
		route.DELETE("/:id", delete, lotsController.DeleteLot)
	}
}
