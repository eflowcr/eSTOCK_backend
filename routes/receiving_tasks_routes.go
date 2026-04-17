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
	"gorm.io/gorm"
)

var _ ports.ReceivingTasksRepository = (*repositories.ReceivingTasksRepository)(nil)

func RegisterReceivingTasksRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, notifSvc *services.NotificationsService) {
	_, receivingTasksService := wire.NewReceivingTasks(db, notifSvc)
	receivingTasksController := controllers.NewReceivingTasksController(*receivingTasksService, config.JWTSecret)

	route := router.Group("/receiving-tasks")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", receivingTasksController.GetAllReceivingTasks)
		route.GET("/:id", receivingTasksController.GetReceivingTaskByID)
		route.POST("/", receivingTasksController.CreateReceivingTask)
		route.PUT("/:id", receivingTasksController.UpdateReceivingTask)
		route.GET("/import/template", receivingTasksController.DownloadImportTemplate)
		route.POST("/import", receivingTasksController.ImportReceivingTaskFromExcel)
		route.GET("/export", receivingTasksController.ExportReceivingTaskToExcel)
		route.PATCH("/complete-full-task/:id/:location", receivingTasksController.CompleteFullTask)
		route.PATCH("/complete-receiving-line/:id/:location", receivingTasksController.CompleteReceivingLine)
	}
}
