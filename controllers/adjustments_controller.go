package controllers

import (
	"encoding/json"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type AdjustmentsController struct {
	Service      services.AdjustmentsService
	JWTSecret    string
	AuditService *services.AuditService
	TenantID     string // S2.5 M3.1 — tenant-scoped reads/writes
}

func NewAdjustmentsController(service services.AdjustmentsService, jwtSecret string, auditSvc *services.AuditService) *AdjustmentsController {
	return &AdjustmentsController{
		Service:      service,
		JWTSecret:    jwtSecret,
		AuditService: auditSvc,
	}
}

// WithTenantID sets the tenant ID (S2.5 M3.1 pattern).
func (c *AdjustmentsController) WithTenantID(tenantID string) *AdjustmentsController {
	c.TenantID = tenantID
	return c
}

func (c *AdjustmentsController) GetAllAdjustments(ctx *gin.Context) {
	adjustments, response := c.Service.ListByTenant(c.resolveTenantID(ctx))

	if response != nil {
		writeErrorResponse(ctx, "GetAllAdjustments", "get_all_adjustments", response)
		return
	}

	if len(adjustments) == 0 {
		tools.ResponseOK(ctx, "GetAllAdjustments", "No adjustments found", "get_all_adjustments", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllAdjustments", "Ajustes obtenidos con éxito", "get_all_adjustments", adjustments, false, "")
}

func (c *AdjustmentsController) GetAdjustmentByID(ctx *gin.Context) {
	adjustmentID, ok := tools.ParseRequiredParam(ctx, "id", "GetAdjustmentByID", "get_adjustment_by_id", "El ID proporcionado no es válido")
	if !ok {
		return
	}

	adjustment, response := c.Service.GetAdjustmentByID(adjustmentID)
	if response != nil {
		writeErrorResponse(ctx, "GetAdjustmentByID", "get_adjustment_by_id", response)
		return
	}

	if adjustment == nil {
		tools.ResponseNotFound(ctx, "GetAdjustmentByID", "Ajuste no encontrado", "get_adjustment_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetAdjustmentByID", "Ajuste obtenido con éxito", "get_adjustment_by_id", adjustment, false, "")
}

func (c *AdjustmentsController) GetAdjustmentDetails(ctx *gin.Context) {
	adjustmentID, ok := tools.ParseRequiredParam(ctx, "id", "GetAdjustmentDetails", "get_adjustment_details", "El ID proporcionado no es válido")
	if !ok {
		return
	}

	details, response := c.Service.GetAdjustmentDetails(adjustmentID)
	if response != nil {
		writeErrorResponse(ctx, "GetAdjustmentDetails", "get_adjustment_details", response)
		return
	}

	if details == nil {
		tools.ResponseNotFound(ctx, "GetAdjustmentDetails", "Detalles del ajuste no encontrados", "get_adjustment_details")
		return
	}

	tools.ResponseOK(ctx, "GetAdjustmentDetails", "Detalles del ajuste obtenidos con éxito", "get_adjustment_details", details, false, "")
}

func (c *AdjustmentsController) CreateAdjustment(ctx *gin.Context) {
	var adjustment requests.CreateAdjustment

	if err := ctx.ShouldBindJSON(&adjustment); err != nil {
		tools.ResponseBadRequest(ctx, "CreateAdjustment", "Carga útil de solicitud no válida", "create_adjustment")
		return
	}
	if errs := tools.ValidateStruct(&adjustment); errs != nil {
		tools.ResponseValidationError(ctx, "CreateAdjustment", "create_adjustment", errs)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	created, response := c.Service.CreateAdjustment(userId, c.resolveTenantID(ctx), adjustment)
	if response != nil {
		writeErrorResponse(ctx, "CreateAdjustment", "create_adjustment", response)
		return
	}

	if c.AuditService != nil && created != nil {
		newVal, _ := json.Marshal(map[string]interface{}{
			"id":       created.ID,
			"sku":      created.SKU,
			"location": created.Location,
			"quantity": created.AdjustmentQty,
			"reason":   created.Reason,
			"user_id":  created.UserID,
		})
		c.AuditService.Log(ctx.Request.Context(), &userId, tools.ActionCreate, tools.ResourceAdjustment, created.ID, nil, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}

	tools.ResponseCreated(ctx, "CreateAdjustment", "Ajuste creado con éxito", "create_adjustment", created, false, "")
}

func (c *AdjustmentsController) ExportAdjustmentsToExcel(ctx *gin.Context) {
	data, response := c.Service.ExportAdjustmentsToExcel(c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "ExportAdjustmentsToExcel", "export_adjustments_to_excel", response)
		return
	}

	if data == nil {
		tools.ResponseOK(ctx, "ExportAdjustmentsToExcel", "No hay ajustes para exportar", "export_adjustments_to_excel", nil, false, "")
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=adjustments.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *AdjustmentsController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
