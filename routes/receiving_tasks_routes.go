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

var _ ports.ReceivingTasksRepository = (*repositories.ReceivingTasksRepository)(nil)

func RegisterReceivingTasksRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, notifSvc *services.NotificationsService, pool *pgxpool.Pool, rolesRepo ports.RolesRepository) {
	_, clientsSvc := wire.NewClients(pool)
	_, receivingTasksService := wire.NewReceivingTasks(db, notifSvc)
	if clientsSvc != nil {
		receivingTasksService.WithClientsService(clientsSvc)
	}
	receivingTasksController := controllers.NewReceivingTasksController(*receivingTasksService, config.JWTSecret).
		WithTenantID(config.TenantID)

	route := router.Group("/receiving-tasks")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "receiving_tasks", "read")
		create := tools.RequirePermission(rolesRepo, "receiving_tasks", "create")
		update := tools.RequirePermission(rolesRepo, "receiving_tasks", "update")

		route.GET("/", read, receivingTasksController.GetAllReceivingTasks)
		route.GET("/:id", read, receivingTasksController.GetReceivingTaskByID)
		route.POST("/", create, receivingTasksController.CreateReceivingTask)
		route.PUT("/:id", update, receivingTasksController.UpdateReceivingTask)
		route.GET("/import/template", read, receivingTasksController.DownloadImportTemplate)
		route.POST("/import", create, receivingTasksController.ImportReceivingTaskFromExcel)
		route.GET("/export", read, receivingTasksController.ExportReceivingTaskToExcel)
		route.PATCH("/complete-full-task/:id/:location", update, receivingTasksController.CompleteFullTask)
		route.PATCH("/complete-receiving-line/:id/:location", update, receivingTasksController.CompleteReceivingLine)
		route.PATCH("/:id/supplier", update, receivingTasksController.LinkSupplier) // S2 R2 E1.7
	}
}
