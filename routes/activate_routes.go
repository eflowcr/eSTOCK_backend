package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, config configuration.Config) {
	RegisterHealthRoutes(r, db)

	api := r.Group("/api")

	RegisterAuthenticationRoutes(api, db, config)
	RegisterEncryptionRoutes(api, config)
	RegisterUserRoutes(api, db, config)
	RegisterDashboardRoutes(api, db, config)
	RegisterLocationRoutes(api, db, config)
	RegisterArticlesRoutes(api, db, config)
	RegisterInventoryRoutes(api, db, config)
	RegisterLotsRoutes(api, db, config)
	RegisterSerialRoutes(api, db, config)
	RegisterReceivingTasksRoutes(api, db, config)
	RegisterPickingTasksRoutes(api, db, config)
	RegisterAdjustmentsRoutes(api, db, config)
	RegisterStockAlertsRoutes(api, db, config)
	RegisterInventoryMovementsRoutes(api, db, config)
	RegisterGamificationRoutes(api, db, config)
	RegisterPresentationsRoutes(api, db, config)
}
