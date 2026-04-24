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

var _ ports.PickingTaskRepository = (*repositories.PickingTaskRepository)(nil)

func RegisterPickingTasksRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, auditSvc *services.AuditService, notifSvc *services.NotificationsService, pool *pgxpool.Pool, rolesRepo ports.RolesRepository) {
	_, clientsSvc := wire.NewClients(pool)
	// SO3 — inject SalesOrdersRepository so CompletePickingTask can update SO picked quantities.
	soRepo, _ := wire.NewSalesOrders(db, config)
	_, pickingTasksService := wire.NewPickingTask(db, auditSvc, notifSvc, soRepo)
	if clientsSvc != nil {
		pickingTasksService.WithClientsService(clientsSvc)
	}
	pickingTasksController := controllers.NewPickingTasksController(*pickingTasksService, config.JWTSecret).
		WithTenantID(config.TenantID)

	route := router.Group("/picking-tasks")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "picking_tasks", "read")
		create := tools.RequirePermission(rolesRepo, "picking_tasks", "create")
		update := tools.RequirePermission(rolesRepo, "picking_tasks", "update")

		route.GET("/", read, pickingTasksController.GetAllPickingTasks)
		route.GET("/:id", read, pickingTasksController.GetPickingTaskByID)
		route.POST("/", create, pickingTasksController.CreatePickingTask)
		route.PUT("/:id", update, pickingTasksController.UpdatePickingTask)
		route.PATCH("/:id/start", update, pickingTasksController.StartPickingTask)
		route.PATCH("/:id/cancel", update, pickingTasksController.CancelPickingTask)
		route.PATCH("/:id/complete", update, pickingTasksController.CompletePickingTask)
		route.PATCH("/:id/complete-line", update, pickingTasksController.CompletePickingLine)
		route.GET("/import/template", read, pickingTasksController.DownloadImportTemplate)
		route.POST("/import", create, pickingTasksController.ImportPickingTaskFromExcel)
		route.GET("/export", read, pickingTasksController.ExportPickingTasksToExcel)
		route.PATCH("/:id/customer", update, pickingTasksController.LinkCustomer) // S2 R2 E1.7
	}
}
