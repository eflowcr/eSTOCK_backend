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
)

var _ ports.AdjustmentReasonCodesRepository = (*repositories.AdjustmentReasonCodesRepositorySQLC)(nil)

func RegisterAdjustmentReasonCodesRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	if pool == nil {
		return
	}
	_, adjustmentReasonCodesService := wire.NewAdjustmentReasonCodes(pool)
	if adjustmentReasonCodesService == nil {
		return
	}
	ctrl := controllers.NewAdjustmentReasonCodesController(*adjustmentReasonCodesService)

	route := router.Group("/adjustment-reason-codes")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readInventory := tools.RequirePermission(rolesRepo, "inventory", "read")
		updateInventory := tools.RequirePermission(rolesRepo, "inventory", "update")

		route.GET("/", readInventory, ctrl.ListAdjustmentReasonCodes)
		route.GET("/admin", updateInventory, ctrl.ListAdjustmentReasonCodesAdmin)
		route.GET("/:id", readInventory, ctrl.GetAdjustmentReasonCodeByID)
		route.POST("/", updateInventory, ctrl.CreateAdjustmentReasonCode)
		route.PUT("/:id", updateInventory, ctrl.UpdateAdjustmentReasonCode)
		route.DELETE("/:id", updateInventory, ctrl.DeleteAdjustmentReasonCode)
	}
}
