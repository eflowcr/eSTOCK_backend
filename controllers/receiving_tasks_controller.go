package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type ReceivingTasksController struct {
	Service   services.ReceivingTasksService
	JWTSecret string
	TenantID  string // S2 R2 — used for supplier validation context
}

func NewReceivingTasksController(service services.ReceivingTasksService, jwtSecret string) *ReceivingTasksController {
	return &ReceivingTasksController{
		Service:   service,
		JWTSecret: jwtSecret,
	}
}

// WithTenantID sets the tenant ID (S2 R2 pattern).
func (c *ReceivingTasksController) WithTenantID(tenantID string) *ReceivingTasksController {
	c.TenantID = tenantID
	return c
}

func (c *ReceivingTasksController) GetAllReceivingTasks(ctx *gin.Context) {
	tasks, response := c.Service.ListByTenant(c.resolveTenantID(ctx))

	if response != nil {
		writeErrorResponse(ctx, "GetAllReceivingTasks", "get_all_receiving_tasks", response)
		return
	}

	if len(tasks) == 0 {
		tools.ResponseOK(ctx, "GetAllReceivingTasks", "No se encontraron tareas de recepción", "get_all_receiving_tasks", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllReceivingTasks", "Tareas de recepción obtenidas con éxito", "get_all_receiving_tasks", tasks, false, "")
}

func (c *ReceivingTasksController) GetReceivingTaskByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetReceivingTaskByID", "get_receiving_task_by_id", "ID de tarea inválido")
	if !ok {
		return
	}

	task, response := c.Service.GetReceivingTaskByID(id)
	if response != nil {
		writeErrorResponse(ctx, "GetReceivingTaskByID", "get_receiving_task_by_id", response)
		return
	}

	if task == nil {
		tools.ResponseNotFound(ctx, "GetReceivingTaskByID", "Tarea de recepción no encontrada", "get_receiving_task_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetReceivingTaskByID", "Tarea de recepción obtenida con éxito", "get_receiving_task_by_id", task, false, "")
}

func (c *ReceivingTasksController) CreateReceivingTask(ctx *gin.Context) {
	var request requests.CreateReceivingTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		tools.ResponseBadRequest(ctx, "CreateReceivingTask", "Formato de solicitud inválido", "create_receiving_task")
		return
	}
	if errs := tools.ValidateStruct(&request); errs != nil {
		tools.ResponseValidationError(ctx, "CreateReceivingTask", "create_receiving_task", errs)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}
	response := c.Service.CreateReceivingTask(userId, c.resolveTenantID(ctx), &request)

	if response != nil {
		writeErrorResponse(ctx, "CreateReceivingTask", "create_receiving_task", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateReceivingTask", "Tarea de recepción creada con éxito", "create_receiving_task", nil, false, "")
}

func (c *ReceivingTasksController) UpdateReceivingTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateReceivingTask", "patch_receiving_task", "ID de tarea inválido")
	if !ok {
		return
	}

	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "PatchReceivingTask", "Formato de cuerpo de solicitud inválido", "patch_receiving_task")
		return
	}

	resp := c.Service.UpdateReceivingTask(id, data)
	if resp != nil {
		writeErrorResponse(ctx, "PatchReceivingTask", "patch_receiving_task", resp)
		return
	}

	tools.ResponseOK(ctx, "PatchReceivingTask", "Tarea de recepción actualizada con éxito", "patch_receiving_task", nil, false, "")
}

func (c *ReceivingTasksController) ImportReceivingTaskFromExcel(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportReceivingTaskFromExcel", "Error al subir el archivo", "import_receiving_task_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportReceivingTaskFromExcel", "Error al abrir el archivo", "import_receiving_task_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportReceivingTaskFromExcel", "Error al leer el contenido del archivo", "import_receiving_task_from_excel")
		return
	}

	response := c.Service.ImportReceivingTaskFromExcel(userId, c.resolveTenantID(ctx), fileBytes)
	if response != nil {
		writeErrorResponse(ctx, "ImportReceivingTaskFromExcel", "import_receiving_task_from_excel", response)
		return
	}

	tools.ResponseOK(ctx, "ImportReceivingTaskFromExcel", "Tareas de recepción importadas con éxito", "import_receiving_task_from_excel", nil, false, "")
}

func (c *ReceivingTasksController) DownloadImportTemplate(ctx *gin.Context) {
	lang := ctx.DefaultQuery("lang", "es")
	data, err := c.Service.GenerateImportTemplate(lang)
	if err != nil {
		tools.ResponseBadRequest(ctx, "DownloadImportTemplate", "Error al generar la plantilla", "download_import_template")
		return
	}
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="ImportReceivingTasks.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (c *ReceivingTasksController) ExportReceivingTaskToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportReceivingTaskToExcel(c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "ExportReceivingTaskToExcel", "export_receiving_task_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="receiving_tasks.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ReceivingTasksController) CompleteFullTask(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CompleteFullTask", "complete_full_task", "ID de tarea inválido")
	if !ok {
		return
	}

	location := ctx.Param("location")
	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	response := c.Service.CompleteFullTask(id, location, userId)
	if response != nil {
		writeErrorResponse(ctx, "CompleteFullTask", "complete_full_task", response)
		return
	}

	tools.ResponseOK(ctx, "CompleteFullTask", "Tarea de recepción marcada como completa con éxito", "complete_full_task", nil, false, "")
}

func (c *ReceivingTasksController) CompleteReceivingLine(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "CompleteReceivingLine", "complete_receiving_line", "ID de tarea inválido")
	if !ok {
		return
	}

	location := ctx.Param("location")

	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	var item requests.ReceivingTaskItemRequest
	if err := ctx.ShouldBindJSON(&item); err != nil {
		tools.ResponseBadRequest(ctx, "CompleteReceivingLine", "Formato de solicitud inválido", "complete_receiving_line")
		return
	}
	if errs := tools.ValidateStruct(&item); errs != nil {
		tools.ResponseValidationError(ctx, "CompleteReceivingLine", "complete_receiving_line", errs)
		return
	}

	response := c.Service.CompleteReceivingLine(id, location, userId, item)
	if response != nil {
		writeErrorResponse(ctx, "CompleteReceivingLine", "complete_receiving_line", response)
		return
	}

	tools.ResponseOK(ctx, "CompleteReceivingLine", "Línea de recepción marcada como completa con éxito", "complete_receiving_line", nil, false, "")
}

// LinkSupplier handles PATCH /receiving-tasks/:id/supplier (S2 R2 E1.7).
func (c *ReceivingTasksController) LinkSupplier(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "LinkSupplier", "link_supplier", "ID de tarea inválido")
	if !ok {
		return
	}

	var req requests.LinkSupplierRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "LinkSupplier", "Formato de solicitud inválido", "link_supplier")
		return
	}

	resp := c.Service.LinkSupplier(id, req.SupplierID)
	if resp != nil {
		writeErrorResponse(ctx, "LinkSupplier", "link_supplier", resp)
		return
	}

	if req.SupplierID == nil || *req.SupplierID == "" {
		tools.ResponseOK(ctx, "LinkSupplier", "Proveedor desvinculado de la tarea", "link_supplier", nil, false, "")
	} else {
		tools.ResponseOK(ctx, "LinkSupplier", "Proveedor vinculado a la tarea", "link_supplier", nil, false, "")
	}
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *ReceivingTasksController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
