package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterAuditRoutes registers GET /api/audit-logs. Requires JWT + audit_logs:read (admin only by default).
func RegisterAuditRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, auditSvc *services.AuditService, rolesRepo ports.RolesRepository) {
	if auditSvc == nil {
		return
	}
	ctrl := controllers.NewAuditController(auditSvc)
	route := router.Group("/audit-logs")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", tools.RequirePermission(rolesRepo, "audit_logs", "read"), ctrl.ListAuditLogs)
	}
}
