package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	api := r.Group("/api")

	RegisterAuthenticationRoutes(api, db)
	RegisterEncryptionRoutes(api)
	RegisterUserRoutes(api, db)
	RegisterDashboardRoutes(api, db)
	RegisterLocationRoutes(api, db)
	RegisterArticlesRoutes(api, db)
	RegisterInventoryRoutes(api, db)
	RegisterLotsRoutes(api, db)
	RegisterSerialRoutes(api, db)
	RegisterReceivingTasksRoutes(api, db)
	RegisterPickingTasksRoutes(api, db)
	RegisterAdjustmentsRoutes(api, db)
	RegisterStockAlertsRoutes(api, db)
}
