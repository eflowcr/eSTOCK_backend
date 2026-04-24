package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterDeliveryNotesRoutes wires /api/delivery-notes endpoints (DN3).
func RegisterDeliveryNotesRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	if db == nil {
		return
	}
	_, svc := wire.NewDeliveryNotes(db)
	ctrl := controllers.NewDeliveryNotesController(svc, config.TenantID)

	route := router.Group("/delivery-notes")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "delivery_notes", "read")

		// DN3 — list + detail + PDF download
		route.GET("", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		// NOTE: /:id/pdf must be registered before /:id to avoid Gin ambiguity.
		// Use separate route group to avoid conflicts.
		route.GET("/:id/pdf", read, ctrl.DownloadPDF)
	}
}
