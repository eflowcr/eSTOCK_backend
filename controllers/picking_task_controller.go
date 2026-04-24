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
	TenantID  string // S2 R2
}

func NewPickingTasksController(service services.PickingTaskService, jwtSecret string) *PickingTasksController {
	return &PickingTasksController{Service: service, JWTSecret: jwtSecret}
}

// WithTenantID sets the tenant ID (S2 R2 pattern).
func (c *PickingTasksController) WithTenantID(tenantID string) *PickingTasksController {
	c.TenantID = tenantID
	return c
}

func (c *PickingTasksController) GetAllPickingTasks(ctx *gin.Context) {
	tasks, response := c.Service.ListByTenant(c.resolveTenantID(ctx))
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

	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	response := c.Service.CreatePickingTask(userId, c.resolveTenantID(ctx), &request)
	if response != nil {
		writeErrorResponse(ctx, "CreatePickingTask", "create_picking_task", response)
		return
	}
	tools.ResponseCreated(ctx, "CreatePickingTask", "Tarea de picking creada con éxito", "create_picking_task", nil, false, "")
}

// StartPickingTask transitions the task to in_progress and applies lazy reservations (B3a).
func (c *PickingTasksController) StartPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "StartPickingTask", "start_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	if resp := c.Service.StartPickingTask(ctx.Request.Context(), id, userId); resp != nil {
		writeErrorResponse(ctx, "StartPickingTask", "start_picking_task", resp)
		return
	}
	tools.ResponseOK(ctx, "StartPickingTask", "Picking iniciado — stock reservado", "start_picking_task", nil, false, "")
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

	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	response := c.Service.UpdatePickingTask(ctx.Request.Context(), id, data, userId)
	if response != nil {
		writeErrorResponse(ctx, "UpdatePickingTask", "update_picking_task", response)
		return
	}
	tools.ResponseOK(ctx, "UpdatePickingTask", "Tarea de picking actualizada con éxito", "update_picking_task", nil, false, "")
}

// CancelPickingTask sets status to cancelled (B3c).
func (c *PickingTasksController) CancelPickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CancelPickingTask", "cancel_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	resp := c.Service.UpdatePickingTask(ctx.Request.Context(), id, map[string]interface{}{"status": "cancelled"}, userId)
	if resp != nil {
		writeErrorResponse(ctx, "CancelPickingTask", "cancel_picking_task", resp)
		return
	}
	tools.ResponseOK(ctx, "CancelPickingTask", "Tarea de picking cancelada con éxito", "cancel_picking_task", nil, false, "")
}

// CompletePickingTask finalises all items using their allocations (H5).
// The old :location URL parameter is ignored — locations come from item allocations.
func (c *PickingTasksController) CompletePickingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CompletePickingTask", "complete_picking_task", "ID de tarea inválido")
	if !ok {
		return
	}
	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	response := c.Service.CompletePickingTask(ctx.Request.Context(), id, userId)
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

	var item requests.PickingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.ResponseBadRequest(ctx, "CompletePickingLine", "Datos de solicitud inválidos", "complete_picking_line")
		return
	}
	if errs := tools.ValidateStruct(&item); errs != nil {
		tools.ResponseValidationError(ctx, "CompletePickingLine", "complete_picking_line", errs)
		return
	}

	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	response := c.Service.CompletePickingLine(ctx.Request.Context(), id, userId, item)
	if response != nil {
		writeErrorResponse(ctx, "CompletePickingLine", "complete_picking_line", response)
		return
	}
	tools.ResponseOK(ctx, "CompletePickingLine", "Línea de picking completada con éxito", "complete_picking_line", nil, false, "")
}

func (c *PickingTasksController) ImportPickingTaskFromExcel(ctx *gin.Context) {
	userId, err := tools.GetUserId(c.JWTSecret, ctx.Request.Header.Get("Authorization"))
	if err != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

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

	response := c.Service.ImportPickingTaskFromExcel(userId, c.resolveTenantID(ctx), fileBytes)
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
	fileBytes, response := c.Service.ExportPickingTasksToExcel(c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "ExportPickingTasksToExcel", "export_picking_tasks_to_excel", response)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename=picking_tasks.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

// LinkCustomer handles PATCH /picking-tasks/:id/customer (S2 R2 E1.7).
func (c *PickingTasksController) LinkCustomer(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "LinkCustomer", "link_customer", "ID de tarea inválido")
	if !ok {
		return
	}

	var req requests.LinkCustomerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "LinkCustomer", "Formato de solicitud inválido", "link_customer")
		return
	}

	resp := c.Service.LinkCustomer(id, req.CustomerID)
	if resp != nil {
		writeErrorResponse(ctx, "LinkCustomer", "link_customer", resp)
		return
	}

	if req.CustomerID == nil || *req.CustomerID == "" {
		tools.ResponseOK(ctx, "LinkCustomer", "Cliente desvinculado de la tarea", "link_customer", nil, false, "")
	} else {
		tools.ResponseOK(ctx, "LinkCustomer", "Cliente vinculado a la tarea", "link_customer", nil, false, "")
	}
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *PickingTasksController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
