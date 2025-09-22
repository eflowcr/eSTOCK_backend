package controllers

import (
	"io"
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

func (c *ReceivingTasksController) ImportReceivingTaskFromExcel(ctx *gin.Context) {
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

	response := c.Service.ImportReceivingTaskFromExcel(userId, fileBytes)
	if response != nil {
		tools.Response(ctx, "ImportReceivingTaskFromExcel", false, response.Message, "import_receiving_task_from_excel", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "ImportReceivingTaskFromExcel", true, "Receiving tasks imported successfully", "import_receiving_task_from_excel", nil, false, "")
}

func (c *ReceivingTasksController) ExportReceivingTaskToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportReceivingTaskToExcel()
	if response != nil {
		tools.Response(ctx, "ExportReceivingTaskToExcel", false, response.Message, "export_receiving_task_to_excel", nil, response.Handled, "")
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="receiving_tasks.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ReceivingTasksController) CompleteFullTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompleteFullTask", false, "Invalid task ID", "complete_full_task", nil, false, "")
		return
	}

	location := ctx.Param("location")
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CompleteFullTask(id, location, userId)
	if response != nil {
		tools.Response(ctx, "CompleteFullTask", false, response.Message, "complete_full_task", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "CompleteFullTask", true, "Receiving task marked as complete successfully", "complete_full_task", nil, false, "")
}

func (c *ReceivingTasksController) CompleteReceivingLine(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompleteReceivingLine", false, "Invalid task ID", "complete_receiving_line", nil, false, "")
		return
	}

	location := ctx.Param("location")

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	var item requests.ReceivingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.Response(ctx, "CompleteReceivingLine", false, "Invalid request format", "complete_receiving_line", nil, true, "")
		return
	}

	response := c.Service.CompleteReceivingLine(id, location, userId, item)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "CompleteReceivingLine", true, response.Message, "complete_receiving_line", nil, true, "")
			return
		}

		tools.Response(ctx, "CompleteReceivingLine", false, response.Message, "complete_receiving_line", nil, response.Handled, "")
		return
	}

	tools.Response(ctx, "CompleteReceivingLine", true, "Receiving line marked as complete successfully", "complete_receiving_line", nil, false, "")
}
