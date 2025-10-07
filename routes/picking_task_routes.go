package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterPickingTasksRoutes(router *gin.RouterGroup, db *gorm.DB) {
	pickingTaskRepository := &repositories.PickingTaskRepository{DB: db}
	pickingTasksService := services.NewPickingTaskService(pickingTaskRepository)
	pickingTasksController := controllers.NewPickingTasksController(*pickingTasksService)

	route := router.Group("/picking-tasks")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", pickingTasksController.GetAllPickingTasks)
		route.GET("/:id", pickingTasksController.GetPickingTaskByID)
		route.POST("/", pickingTasksController.CreatePickingTask)
		route.PUT("/:id", pickingTasksController.UpdatePickingTask)
		route.POST("/import", pickingTasksController.ImportPickingTaskFromExcel)
		route.GET("/export", pickingTasksController.ExportPickingTasksToExcel)
		route.PATCH("/complete-full-task/:id/:location", pickingTasksController.CompletePickingTask)
		route.PATCH("/complete-picking-line/:id/:location", pickingTasksController.CompletePickingLine)
	}
}
