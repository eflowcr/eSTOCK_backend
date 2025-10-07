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
		tools.Response(ctx, "GetAllPickingTasks", false, response.Message, "get_all_picking_tasks", nil, false, "", response.Handled)
		return
	}

	if len(tasks) == 0 {
		tools.Response(ctx, "GetAllPickingTasks", true, "No se encontraron tareas de picking", "get_all_picking_tasks", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllPickingTasks", true, "Tareas de picking recuperadas con éxito", "get_all_picking_tasks", tasks, false, "", false)
}

func (c *PickingTasksController) GetPickingTaskByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil || id <= 0 {
		tools.Response(ctx, "GetPickingTaskByID", false, "ID de tarea inválido", "get_picking_task_by_id", nil, false, "", true)

		return
	}

	task, response := c.Service.GetPickingTaskByID(id)

	if response != nil {
		tools.Response(ctx, "GetPickingTaskByID", false, response.Message, "get_picking_task_by_id", nil, false, "", response.Handled)
		return
	}

	if task == nil {
		tools.Response(ctx, "GetPickingTaskByID", true, "Tarea de picking no encontrada", "get_picking_task_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetPickingTaskByID", true, "Tarea de picking recuperada con éxito", "get_picking_task_by_id", task, false, "", false)
}

func (c *PickingTasksController) CreatePickingTask(ctx *gin.Context) {
	var request requests.CreatePickingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.Response(ctx, "CreatePickingTask", false, "Cuerpo de solicitud inválido", "create_picking_task", nil, false, "", false)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CreatePickingTask(userId, &request)
	if response != nil {
		if response.Handled {
			tools.Response(ctx, "CreatePickingTask", true, response.Message, "create_picking_task", nil, true, "", false)

			return
		}

		tools.Response(ctx, "CreatePickingTask", false, response.Message, "create_picking_task", nil, response.Handled, "", false)

		return
	}

	tools.Response(ctx, "CreatePickingTask", true, "Tarea de picking creada con éxito", "create_picking_task", nil, false, "", false)
}

func (c *PickingTasksController) UpdatePickingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "UpdatePickingTask", false, "ID de tarea inválido", "update_picking_task", nil, false, "", false)
		return
	}

	var data map[string]interface{}

	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdatePickingTask", false, "Cuerpo de solicitud inválido", "update_picking_task", nil, false, "", false)
		return
	}

	response := c.Service.UpdatePickingTask(id, data)
	if response != nil {
		tools.Response(ctx, "UpdatePickingTask", false, response.Message, "update_picking_task", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "UpdatePickingTask", true, "Tarea de picking actualizada con éxito", "update_picking_task", nil, false, "", false)
}

func (c *PickingTasksController) ImportPickingTaskFromExcel(ctx *gin.Context) {
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

	response := c.Service.ImportPickingTaskFromExcel(userId, fileBytes)
	if response != nil {
		tools.Response(ctx, "ImportLocationsFromExcel", false, response.Message, "import_locations_from_excel", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "ImportLocationsFromExcel", true, "Tareas de picking importadas con éxito", "import_locations_from_excel", nil, false, "", false)
}

func (c *PickingTasksController) ExportPickingTasksToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportPickingTasksToExcel()
	if response != nil {
		tools.Response(ctx, "ExportPickingTasksToExcel", false, response.Message, "export_picking_tasks_to_excel", nil, response.Handled, "", false)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=picking_tasks.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *PickingTasksController) CompletePickingTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompletePickingTask", false, "ID de tarea inválido", "complete_picking_task", nil, false, "", true)
		return
	}

	location := ctx.Param("location")

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CompletePickingTask(id, location, userId)
	if response != nil {
		tools.Response(ctx, "CompletePickingTask", false, response.Message, "complete_picking_task", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "CompletePickingTask", true, "Tarea de picking completada con éxito", "complete_picking_task", nil, false, "", false)
}

func (c *PickingTasksController) CompletePickingLine(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		tools.Response(ctx, "CompletePickingLine", false, "ID de tarea inválido", "complete_picking_line", nil, false, "", true)
		return
	}

	location := ctx.Param("location")

	var item requests.PickingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.Response(ctx, "CompletePickingLine", false, "Datos de solicitud inválidos", "complete_picking_line", nil, false, "", true)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CompletePickingLine(id, location, userId, item)

	if response != nil {
		if response.Handled {
			tools.Response(ctx, "CompletePickingLine", true, response.Message, "complete_picking_line", nil, true, "", false)
		}

		tools.Response(ctx, "CompletePickingLine", false, response.Message, "complete_picking_line", nil, response.Handled, "", false)
		return
	}

	tools.Response(ctx, "CompletePickingLine", true, "Línea de picking completada con éxito", "complete_picking_line", nil, false, "", false)
}
