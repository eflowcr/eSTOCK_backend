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

func RegisterLotsRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) {
	_, lotsService := wire.NewLots(db, pool)
	lotsController := controllers.NewLotsController(*lotsService)

	route := router.Group("/lots")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		// Legacy list endpoint; generic table handler is available at /lots/table.
		route.GET("/", lotsController.GetAllLots)
		if pool != nil {
			cfg := tools.LotsTableConfig()
			route.GET("/table", tools.GenericListHandler(pool, cfg))
			route.GET("/table/export", tools.GenericExportHandler(pool, cfg, "lots.csv"))
		}
		route.GET("/:sku", lotsController.GetLotsBySKU)
		route.POST("/", lotsController.CreateLot)
		route.PUT("/:id", lotsController.UpdateLot)
		route.DELETE("/:id", lotsController.DeleteLot)
	}
}
