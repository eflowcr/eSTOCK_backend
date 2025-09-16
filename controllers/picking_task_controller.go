package controllers

import (
	"io"
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type PickingTasksController struct {
	Service services.PickingTaskService
}

func NewPickingTasksController(service services.PickingTaskService) *PickingTasksController {
	return &PickingTasksController{
		Service: service,
	}
}

func (c *PickingTasksController) GetAllPickingTasks(ctx *gin.Context) {
	tasks, response := c.Service.GetAllPickingTasks()

	if response != nil {
		tools.Response(ctx, "GetAllPickingTasks", false, response.Message, "get_all_picking_tasks", nil, false, "")
		return
	}

	if len(tasks) == 0 {
		tools.Response(ctx, "GetAllPickingTasks", true, "No picking tasks found", "get_all_picking_tasks", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllPickingTasks", true, "Picking tasks retrieved successfully", "get_all_picking_tasks", tasks, false, "")
}

func (c *PickingTasksController) GetPickingTaskByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "GetPickingTaskByID", false, "Invalid task ID", "get_picking_task_by_id", nil, false, "")
		return
	}

	task, response := c.Service.GetPickingTaskByID(id)
	if response != nil {
		tools.Response(ctx, "GetPickingTaskByID", false, response.Message, "get_picking_task_by_id", nil, response.Handled, "")
		return
	}

	if task == nil {
		tools.Response(ctx, "GetPickingTaskByID", true, "Picking task not found", "get_picking_task_by_id", nil, false, "")
		return
	}

	tools.Response(ctx, "GetPickingTaskByID", true, "Picking task retrieved successfully", "get_picking_task_by_id", task, false, "")
}

func (c *PickingTasksController) CreatePickingTask(ctx *gin.Context) {
	var request requests.CreatePickingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreatePickingTask", false, "Invalid request data", "create_picking_task", nil, false, "")
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CreatePickingTask(userId, &request)
	if response != nil {
		tools.Response(ctx, "CreatePickingTask", false, response.Message, "create_picking_task", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "CreatePickingTask", true, "Picking task created successfully", "create_picking_task", nil, false, "")
}

func (c *PickingTasksController) UpdatePickingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "UpdatePickingTask", false, "Invalid task ID", "update_picking_task", nil, false, "")
		return
	}

	var data map[string]interface{}

	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdatePickingTask", false, "Invalid request body", "update_picking_task", nil, false, "")
		return
	}

	response := c.Service.UpdatePickingTask(id, data)
	if response != nil {
		tools.Response(ctx, "UpdatePickingTask", false, response.Message, "update_picking_task", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "UpdatePickingTask", true, "Picking task updated successfully", "update_picking_task", nil, false, "")
}

func (c *PickingTasksController) ImportPickingTaskFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "File upload error: "+err.Error(), "import_locations_from_excel", nil, false, "")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Failed to open file: "+err.Error(), "import_locations_from_excel", nil, false, "")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Failed to read file content: "+err.Error(), "import_locations_from_excel", nil, false, "")
		return
	}

	response := c.Service.ImportPickingTaskFromExcel(userId, fileBytes)
	if response != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, response.Message, "import_locations_from_excel", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "ImportLocationsFromExcel", true, "Picking tasks imported successfully", "import_locations_from_excel", nil, false, "")
}

func (c *PickingTasksController) ExportPickingTasksToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportPickingTasksToExcel()
	if response != nil {
		tools.Response(ctx, "ExportPickingTasksToExcel", false, response.Message, "export_picking_tasks_to_excel", nil, response.Handled, "")
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=picking_tasks.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *PickingTasksController) CompletePickingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompletePickingTask", false, "Invalid task ID", "complete_picking_task", nil, false, "")
		return
	}

	location := ctx.Param("location")

	response := c.Service.CompletePickingTask(id, location)
	if response != nil {
		tools.Response(ctx, "CompletePickingTask", false, response.Message, "complete_picking_task", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "CompletePickingTask", true, "Picking task completed successfully", "complete_picking_task", nil, false, "")
}
