package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.PickingTaskRepository = (*repositories.PickingTaskRepository)(nil)

func RegisterPickingTasksRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, pickingTasksService := wire.NewPickingTask(db)
	pickingTasksController := controllers.NewPickingTasksController(*pickingTasksService, config.JWTSecret)

	route := router.Group("/picking-tasks")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", pickingTasksController.GetAllPickingTasks)
		route.GET("/:id", pickingTasksController.GetPickingTaskByID)
		route.POST("/", pickingTasksController.CreatePickingTask)
		route.PUT("/:id", pickingTasksController.UpdatePickingTask)
		route.PATCH("/:id/start", pickingTasksController.StartPickingTask)
		route.PATCH("/:id/cancel", pickingTasksController.CancelPickingTask)
		route.PATCH("/:id/complete", pickingTasksController.CompletePickingTask)
		route.GET("/import/template", pickingTasksController.DownloadImportTemplate)
		route.POST("/import", pickingTasksController.ImportPickingTaskFromExcel)
		route.GET("/export", pickingTasksController.ExportPickingTasksToExcel)
		route.PATCH("/complete-full-task/:id/:location", pickingTasksController.CompletePickingTask)
		route.PATCH("/complete-picking-line/:id/:location", pickingTasksController.CompletePickingLine)
	}
}
