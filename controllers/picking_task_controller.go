package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type PickingTasksController struct {
	Service   services.PickingTaskService
	JWTSecret string
}

func NewPickingTasksController(service services.PickingTaskService, jwtSecret string) *PickingTasksController {
	return &PickingTasksController{
		Service:   service,
		JWTSecret: jwtSecret,
	}
}

func (c *PickingTasksController) GetAllPickingTasks(ctx *gin.Context) {
	tasks, response := c.Service.GetAllPickingTasks()

	if response != nil {
		writeErrorResponse(ctx, "GetAllPickingTasks", "get_all_picking_tasks", response)
		return
	}

	if len(tasks) == 0 {
		tools.ResponseOK(ctx, "GetAllPickingTasks", "No se encontraron tareas de picking", "get_all_picking_tasks", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllPickingTasks", "Tareas de picking recuperadas con éxito", "get_all_picking_tasks", tasks, false, "")
}

func (c *PickingTasksController) GetPickingTaskByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetPickingTaskByID", "get_picking_task_by_id", "ID de tarea inválido")
	if !ok {
		return
	}

	task, response := c.Service.GetPickingTaskByID(id)

	if response != nil {
		writeErrorResponse(ctx, "GetPickingTaskByID", "get_picking_task_by_id", response)
		return
	}

	if task == nil {
		tools.ResponseNotFound(ctx, "GetPickingTaskByID", "Tarea de picking no encontrada", "get_picking_task_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetPickingTaskByID", "Tarea de picking recuperada con éxito", "get_picking_task_by_id", task, false, "")
}

func (c *PickingTasksController) CreatePickingTask(ctx *gin.Context) {
	var request requests.CreatePickingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreatePickingTask", "Cuerpo de solicitud inválido", "create_picking_task")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreatePickingTask", "create_picking_task", errs)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	response := c.Service.CreatePickingTask(userId, &request)
	if response != nil {
		writeErrorResponse(ctx, "CreatePickingTask", "create_picking_task", response)
		return
	}

	tools.ResponseCreated(ctx, "CreatePickingTask", "Tarea de picking creada con éxito", "create_picking_task", nil, false, "")
}

func (c *PickingTasksController) UpdatePickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdatePickingTask", "update_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}

	var data map[string]interface{}

	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePickingTask", "Cuerpo de solicitud inválido", "update_picking_task")
		return
	}

	response := c.Service.UpdatePickingTask(id, data)
	if response != nil {
		writeErrorResponse(ctx, "UpdatePickingTask", "update_picking_task", response)
		return
	}

	tools.ResponseOK(ctx, "UpdatePickingTask", "Tarea de picking actualizada con éxito", "update_picking_task", nil, false, "")
}

func (c *PickingTasksController) ImportPickingTaskFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportPickingTaskFromExcel", "Error al subir el archivo", "import_picking_task_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportPickingTaskFromExcel", "Error al abrir el archivo", "import_picking_task_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportPickingTaskFromExcel", "Error al leer el contenido del archivo", "import_picking_task_from_excel")
		return
	}

	response := c.Service.ImportPickingTaskFromExcel(userId, fileBytes)
	if response != nil {
		writeErrorResponse(ctx, "ImportPickingTaskFromExcel", "import_picking_task_from_excel", response)
		return
	}

	tools.ResponseOK(ctx, "ImportPickingTaskFromExcel", "Tareas de picking importadas con éxito", "import_picking_task_from_excel", nil, false, "")
}

func (c *PickingTasksController) DownloadImportTemplate(ctx *gin.Context) {
	lang := ctx.DefaultQuery("lang", "es")
	data, err := c.Service.GenerateImportTemplate(lang)
	if err != nil {
		tools.ResponseBadRequest(ctx, "DownloadImportTemplate", "Error al generar la plantilla", "download_import_template")
		return
	}
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="ImportPickingTasks.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (c *PickingTasksController) ExportPickingTasksToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportPickingTasksToExcel()
	if response != nil {
		writeErrorResponse(ctx, "ExportPickingTasksToExcel", "export_picking_tasks_to_excel", response)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=picking_tasks.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

// StartPickingTask sets status from open -> in_progress.
func (c *PickingTasksController) StartPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "StartPickingTask", "start_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}

	resp := c.Service.UpdatePickingTask(id, map[string]interface{}{"status": "in_progress"})
	if resp != nil {
		writeErrorResponse(ctx, "StartPickingTask", "start_picking_task", resp)
		return
	}

	tools.ResponseOK(ctx, "StartPickingTask", "Tarea de picking iniciada con éxito", "start_picking_task", nil, false, "")
}

// CancelPickingTask sets status to cancelled from open or in_progress.
func (c *PickingTasksController) CancelPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CancelPickingTask", "cancel_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}

	resp := c.Service.UpdatePickingTask(id, map[string]interface{}{"status": "cancelled"})
	if resp != nil {
		writeErrorResponse(ctx, "CancelPickingTask", "cancel_picking_task", resp)
		return
	}

	tools.ResponseOK(ctx, "CancelPickingTask", "Tarea de picking cancelada con éxito", "cancel_picking_task", nil, false, "")
}

func (c *PickingTasksController) CompletePickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CompletePickingTask", "complete_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}

	location := ctx.Param("location")

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	response := c.Service.CompletePickingTask(id, location, userId)
	if response != nil {
		writeErrorResponse(ctx, "CompletePickingTask", "complete_picking_task", response)
		return
	}

	tools.ResponseOK(ctx, "CompletePickingTask", "Tarea de picking completada con éxito", "complete_picking_task", nil, false, "")
}

func (c *PickingTasksController) CompletePickingLine(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CompletePickingLine", "complete_picking_line", "ID de tarea inválido")
	if !ok {
		return
	}

	location := ctx.Param("location")

	var item requests.PickingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.ResponseBadRequest(ctx, "CompletePickingLine", "Datos de solicitud inválidos", "complete_picking_line")
		return
	}
	if errs := tools.ValidateStruct(&item); errs != nil {
		tools.ResponseValidationError(ctx, "CompletePickingLine", "complete_picking_line", errs)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(c.JWTSecret, token)

	response := c.Service.CompletePickingLine(id, location, userId, item)

	if response != nil {
		writeErrorResponse(ctx, "CompletePickingLine", "complete_picking_line", response)
		return
	}

	tools.ResponseOK(ctx, "CompletePickingLine", "Línea de picking completada con éxito", "complete_picking_line", nil, false, "")
}
