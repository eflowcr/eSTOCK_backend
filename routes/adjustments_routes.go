package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

var _ ports.AdjustmentsRepository = (*repositories.AdjustmentsRepository)(nil)

func RegisterAdjustmentsRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, auditSvc *services.AuditService, rolesRepo ports.RolesRepository) {
	_, adjustmentsService := wire.NewAdjustments(db, pool)
	adjustmentsController := controllers.NewAdjustmentsController(*adjustmentsService, config.JWTSecret, auditSvc).
		WithTenantID(config.TenantID)

	route := router.Group("/adjustments")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "adjustments", "read")
		create := tools.RequirePermission(rolesRepo, "adjustments", "create")

		route.GET("/", read, adjustmentsController.GetAllAdjustments)
		route.GET("/:id", read, adjustmentsController.GetAdjustmentByID)
		route.GET("/:id/details", read, adjustmentsController.GetAdjustmentDetails)
		route.POST("/", create, adjustmentsController.CreateAdjustment)
		route.GET("/export", read, adjustmentsController.ExportAdjustmentsToExcel)
	}
}
