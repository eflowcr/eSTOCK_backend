package controllers

import (
	"strconv"

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

func (c *ReceivingTasksController) GetReceivingTaskByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "GetReceivingTaskByID", false, "Invalid task ID", "get_receiving_task_by_id", nil, false, "")
		return
	}

	task, response := c.Service.GetReceivingTaskByID(id)
	if response != nil {
		tools.Response(ctx, "GetReceivingTaskByID", false, response.Message, "get_receiving_task_by_id", nil, response.Handled, "")
		return
	}

	if task == nil {
		tools.Response(ctx, "GetReceivingTaskByID", true, "Receiving task not found", "get_receiving_task_by_id", nil, false, "")
		return
	}

	tools.Response(ctx, "GetReceivingTaskByID", true, "Receiving task retrieved successfully", "get_receiving_task_by_id", task, false, "")
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

func (c *ReceivingTasksController) UpdateReceivingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "PatchReceivingTask", false, "Invalid task ID", "patch_receiving_task", nil, false, "")
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "PatchReceivingTask", false, "Invalid request body", "patch_receiving_task", nil, false, "")
		return
	}

	resp := c.Service.UpdateReceivingTask(id, data)
	if resp != nil {
		tools.Response(ctx, "PatchReceivingTask", false, resp.Message, "patch_receiving_task", nil, false, "")
		return
	}

	tools.Response(ctx, "PatchReceivingTask", true, "Receiving task updated successfully", "patch_receiving_task", nil, false, "")
}
