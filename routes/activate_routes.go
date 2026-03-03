package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) {
	RegisterHealthRoutes(r, db)

	api := r.Group("/api")

	RegisterAuthenticationRoutes(api, db, config)
	RegisterEncryptionRoutes(api, config)
	RegisterUserRoutes(api, db, config)
	RegisterDashboardRoutes(api, db, config)
	RegisterLocationRoutes(api, db, pool, config)
	RegisterArticlesRoutes(api, db, pool, config)
	RegisterInventoryRoutes(api, db, config)
	RegisterLotsRoutes(api, db, pool, config)
	RegisterSerialRoutes(api, db, pool, config)
	RegisterReceivingTasksRoutes(api, db, config)
	RegisterPickingTasksRoutes(api, db, config)
	RegisterAdjustmentsRoutes(api, db, config)
	RegisterStockAlertsRoutes(api, db, config)
	RegisterInventoryMovementsRoutes(api, db, config)
	RegisterGamificationRoutes(api, db, config)
	RegisterPresentationsRoutes(api, db, pool, config)

	RegisterDocsRoutes(r)
}
