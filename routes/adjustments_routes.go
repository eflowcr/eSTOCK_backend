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

func RegisterAdjustmentsRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, auditSvc *services.AuditService) {
	_, adjustmentsService := wire.NewAdjustments(db, pool)
	adjustmentsController := controllers.NewAdjustmentsController(*adjustmentsService, config.JWTSecret, auditSvc)

	route := router.Group("/adjustments")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", adjustmentsController.GetAllAdjustments)
		route.GET("/:id", adjustmentsController.GetAdjustmentByID)
		route.GET("/:id/details", adjustmentsController.GetAdjustmentDetails)
		route.POST("/", adjustmentsController.CreateAdjustment)
		route.GET("/export", adjustmentsController.ExportAdjustmentsToExcel)
	}
}
