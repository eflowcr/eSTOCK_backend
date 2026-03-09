package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) {
	RegisterHealthRoutes(r, db)

	api := r.Group("/api")

	var rolesRepo ports.RolesRepository
	if pool != nil {
		rolesRepo = wire.NewRoles(pool)
	}
	RegisterAuthenticationRoutes(api, db, config, rolesRepo)
	RegisterEncryptionRoutes(api, config)
	RegisterUserRoutes(api, db, config)
	RegisterPreferencesRoutes(api, pool, config)
	RegisterDashboardRoutes(api, db, config)
	RegisterInventoryRoutes(api, db, config)
	RegisterSerialRoutes(api, db, pool, config)
	RegisterReceivingTasksRoutes(api, db, config)
	RegisterPickingTasksRoutes(api, db, config)
	RegisterAdjustmentsRoutes(api, db, config)
	RegisterStockAlertsRoutes(api, db, config)
	RegisterInventoryMovementsRoutes(api, db, config)
	RegisterGamificationRoutes(api, db, config)
	RegisterPresentationsRoutes(api, db, pool, config)

	var auditSvc *services.AuditService
	if pool != nil {
		_, auditSvc = wire.NewAuditLog(pool)
	}
	RegisterAuditRoutes(api, pool, config, auditSvc, rolesRepo)
	RegisterArticlesRoutes(api, db, pool, config, auditSvc, rolesRepo)
	RegisterLocationRoutes(api, db, pool, config, rolesRepo)
	RegisterLocationTypesRoutes(api, pool, config, rolesRepo)
	RegisterLotsRoutes(api, db, pool, config, rolesRepo)
	RegisterRolesRoutes(api, config, rolesRepo)

	RegisterDocsRoutes(r)
}
