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
	var auditSvc *services.AuditService
	if pool != nil {
		_, auditSvc = wire.NewAuditLog(pool)
	}
	RegisterAuthenticationRoutes(api, db, config, rolesRepo)
	RegisterEncryptionRoutes(api, config)
	RegisterUserRoutes(api, db, config)
	RegisterPreferencesRoutes(api, pool, config)
	RegisterDashboardRoutes(api, db, config)
	RegisterInventoryRoutes(api, db, pool, config)
	RegisterSerialRoutes(api, db, pool, config)
	RegisterReceivingTasksRoutes(api, db, config)
	RegisterPickingTasksRoutes(api, db, config)
	RegisterAdjustmentsRoutes(api, db, pool, config, auditSvc)
	RegisterStockAlertsRoutes(api, db, config)
	RegisterInventoryMovementsRoutes(api, db, config)
	RegisterGamificationRoutes(api, db, config)
	RegisterPresentationsRoutes(api, db, pool, config)
	RegisterAuditRoutes(api, pool, config, auditSvc, rolesRepo)
	RegisterArticlesRoutes(api, db, pool, config, auditSvc, rolesRepo)
	RegisterLocationRoutes(api, db, pool, config, rolesRepo)
	RegisterLocationTypesRoutes(api, pool, config, rolesRepo)
	RegisterPresentationTypesRoutes(api, pool, config, rolesRepo)
	RegisterAdjustmentReasonCodesRoutes(api, pool, config, rolesRepo)
	RegisterPresentationConversionsRoutes(api, pool, config, rolesRepo)
	RegisterStockTransfersRoutes(api, db, pool, config, rolesRepo, auditSvc)
	RegisterLotsRoutes(api, db, pool, config, rolesRepo)
	RegisterRolesRoutes(api, config, rolesRepo)

	RegisterDocsRoutes(r)
}
