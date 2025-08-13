package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type ReceivingTasksController struct {
	Service services.ReceivingTasksService
}

func NewReceivingTasksController(service services.ReceivingTasksService) *ReceivingTasksController {
	return &ReceivingTasksController{
		Service: service,
	}
}

// GetAllReceivingTasks retrieves all receiving tasks
func (c *ReceivingTasksController) GetAllReceivingTasks(ctx *gin.Context) {
	tasks, response := c.Service.GetAllReceivingTasks()

	if response != nil {
		tools.Response(ctx, "GetAllReceivingTasks", false, response.Message, "get_all_receiving_tasks", nil, false, "")
		return
	}

	if len(tasks) == 0 {
		tools.Response(ctx, "GetAllReceivingTasks", true, "No receiving tasks found", "get_all_receiving_tasks", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllReceivingTasks", true, "Receiving tasks retrieved successfully", "get_all_receiving_tasks", tasks, false, "")
}

func (c *ReceivingTasksController) CreateReceivingTask(ctx *gin.Context) {
	var request requests.CreateReceivingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateReceivingTask", false, "Invalid request format", "create_receiving_task", nil, true, "")
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)
	response := c.Service.CreateReceivingTask(userId, &request)

	if response != nil {
		tools.Response(ctx, "CreateReceivingTask", false, response.Message, "create_receiving_task", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "CreateReceivingTask", true, "Receiving task created successfully", "create_receiving_task", nil, false, "")
}
