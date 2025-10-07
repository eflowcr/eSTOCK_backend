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
		tools.Response(ctx, "GetAllReceivingTasks", false, response.Message, "get_all_receiving_tasks", nil, false, "", response.Handled)
		return
	}

	if len(tasks) == 0 {
		tools.Response(ctx, "GetAllReceivingTasks", true, "No se encontraron tareas de recepción", "get_all_receiving_tasks", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllReceivingTasks", true, "Tareas de recepción obtenidas con éxito", "get_all_receiving_tasks", tasks, false, "", false)
}

func (c *ReceivingTasksController) GetReceivingTaskByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "GetReceivingTaskByID", false, "ID de tarea inválido", "get_receiving_task_by_id", nil, false, "", false)
		return
	}

	task, response := c.Service.GetReceivingTaskByID(id)
	if response != nil {
		tools.Response(ctx, "GetReceivingTaskByID", false, response.Message, "get_receiving_task_by_id", nil, response.Handled, "", response.Handled)
		return
	}

	if task == nil {
		tools.Response(ctx, "GetReceivingTaskByID", true, "Tarea de recepción no encontrada", "get_receiving_task_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetReceivingTaskByID", true, "Tarea de recepción obtenida con éxito", "get_receiving_task_by_id", task, false, "", false)
}

func (c *ReceivingTasksController) CreateReceivingTask(ctx *gin.Context) {
	var request requests.CreateReceivingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreateReceivingTask", false, "Formato de solicitud inválido", "create_receiving_task", nil, true, "", false)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)
	response := c.Service.CreateReceivingTask(userId, &request)

	if response != nil {
		if response.Handled {
			tools.Response(ctx, "CreateReceivingTask", false, response.Message, "create_receiving_task", nil, true, "", true)
			return
		}

		tools.Response(ctx, "CreateReceivingTask", false, response.Message, "create_receiving_task", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "CreateReceivingTask", true, "Tarea de recepción creada con éxito", "create_receiving_task", nil, false, "", false)
}

func (c *ReceivingTasksController) UpdateReceivingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "PatchReceivingTask", false, "ID de tarea inválido", "patch_receiving_task", nil, false, "", false)
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "PatchReceivingTask", false, "Formato de cuerpo de solicitud inválido", "patch_receiving_task", nil, false, "", false)
		return
	}

	resp := c.Service.UpdateReceivingTask(id, data)
	if resp != nil {
		tools.Response(ctx, "PatchReceivingTask", false, resp.Message, "patch_receiving_task", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "PatchReceivingTask", true, "Tarea de recepción actualizada con éxito", "patch_receiving_task", nil, false, "", false)
}

func (c *ReceivingTasksController) ImportReceivingTaskFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al subir el archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al abrir el archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, "Error al leer el contenido del archivo: "+err.Error(), "import_locations_from_excel", nil, false, "", false)
		return
	}

	response := c.Service.ImportReceivingTaskFromExcel(userId, fileBytes)
	if response != nil {
		tools.Response(ctx, "ImportReceivingTaskFromExcel", false, response.Message, "import_receiving_task_from_excel", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "ImportReceivingTaskFromExcel", true, "Tareas de recepción importadas con éxito", "import_receiving_task_from_excel", nil, false, "", false)
}

func (c *ReceivingTasksController) ExportReceivingTaskToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportReceivingTaskToExcel()
	if response != nil {
		tools.Response(ctx, "ExportReceivingTaskToExcel", false, response.Message, "export_receiving_task_to_excel", nil, false, "", response.Handled)
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
		tools.Response(ctx, "CompleteFullTask", false, "ID de tarea inválido", "complete_full_task", nil, false, "", false)
		return
	}

	location := ctx.Param("location")
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CompleteFullTask(id, location, userId)
	if response != nil {
		tools.Response(ctx, "CompleteFullTask", false, response.Message, "complete_full_task", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CompleteFullTask", true, "Tarea de recepción marcada como completa con éxito", "complete_full_task", nil, false, "", false)
}

func (c *ReceivingTasksController) CompleteReceivingLine(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompleteReceivingLine", false, "ID de tarea inválido", "complete_receiving_line", nil, false, "", false)
		return
	}

	location := ctx.Param("location")

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	var item requests.ReceivingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.Response(ctx, "CompleteReceivingLine", false, "Formato de solicitud inválido", "complete_receiving_line", nil, true, "", false)
		return
	}

	response := c.Service.CompleteReceivingLine(id, location, userId, item)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "CompleteReceivingLine", true, response.Message, "complete_receiving_line", nil, true, "", false)
			return
		}

		tools.Response(ctx, "CompleteReceivingLine", false, response.Message, "complete_receiving_line", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "CompleteReceivingLine", true, "Línea de recepción marcada como completa con éxito", "complete_receiving_line", nil, false, "", false)
}
