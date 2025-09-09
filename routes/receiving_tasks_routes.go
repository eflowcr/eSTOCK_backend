package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterReceivingTasksRoutes(router *gin.RouterGroup, db *gorm.DB) {
	receivingTaskRepository := &repositories.ReceivingTasksRepository{DB: db}
	receivingTasksService := services.NewReceivingTasksService(receivingTaskRepository)
	receivingTasksController := controllers.NewReceivingTasksController(*receivingTasksService)

	route := router.Group("/receiving-tasks")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", receivingTasksController.GetAllReceivingTasks)
		route.GET("/:id", receivingTasksController.GetReceivingTaskByID)
		route.POST("/", receivingTasksController.CreateReceivingTask)
		route.PUT("/:id", receivingTasksController.UpdateReceivingTask)
		route.POST("/import", receivingTasksController.ImportReceivingTaskFromExcel)
		route.GET("/export", receivingTasksController.ExportReceivingTaskToExcel)
		route.PATCH("/complete-full-task/:id/:location", receivingTasksController.CompleteFullTask)
	}
}
