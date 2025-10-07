package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type AdjustmentsController struct {
	Service services.AdjustmentsService
}

func NewAdjustmentsController(service services.AdjustmentsService) *AdjustmentsController {
	return &AdjustmentsController{
		Service: service,
	}
}

func (c *AdjustmentsController) GetAllAdjustments(ctx *gin.Context) {
	adjustments, response := c.Service.GetAllAdjustments()

	if response != nil {
		tools.Response(ctx, "GetAllAdjustments", false, response.Message, "get_all_adjustments", nil, false, "", response.Handled)
		return
	}

	if len(adjustments) == 0 {
		tools.Response(ctx, "GetAllAdjustments", true, "No adjustments found", "get_all_adjustments", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllAdjustments", true, "Ajustes obtenidos con éxito", "get_all_adjustments", adjustments, false, "", false)
}

func (c *AdjustmentsController) GetAdjustmentByID(ctx *gin.Context) {
	id := ctx.Param("id")

	adjustmentId, err := strconv.Atoi(id)
	if err != nil {
		tools.Response(ctx, "GetAdjustmentByID", false, "El ID proporcionado no es válido", "get_adjustment_by_id", nil, false, "", true)
		return
	}

	adjustment, response := c.Service.GetAdjustmentByID(adjustmentId)
	if response != nil {
		tools.Response(ctx, "GetAdjustmentByID", false, response.Message, "get_adjustment_by_id", nil, false, "", response.Handled)
		return
	}

	if adjustment == nil {
		tools.Response(ctx, "GetAdjustmentByID", true, "Ajuste no encontrado", "get_adjustment_by_id", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetAdjustmentByID", true, "Ajuste obtenido con éxito", "get_adjustment_by_id", adjustment, false, "", false)
}

func (c *AdjustmentsController) GetAdjustmentDetails(ctx *gin.Context) {
	id := ctx.Param("id")

	adjustmentId, err := strconv.Atoi(id)

	if err != nil {
		tools.Response(ctx, "GetAdjustmentDetails", false, "El ID proporcionado no es válido", "get_adjustment_details", nil, false, "", true)
		return
	}

	details, response := c.Service.GetAdjustmentDetails(adjustmentId)
	if response != nil {
		tools.Response(ctx, "GetAdjustmentDetails", false, response.Message, "get_adjustment_details", nil, false, "", response.Handled)
		return
	}

	if details == nil {
		tools.Response(ctx, "GetAdjustmentDetails", true, "Detalles del ajuste no encontrados", "get_adjustment_details", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetAdjustmentDetails", true, "Detalles del ajuste obtenidos con éxito", "get_adjustment_details", details, false, "", false)
}

func (c *AdjustmentsController) CreateAdjustment(ctx *gin.Context) {
	var adjustment requests.CreateAdjustment

	if err := ctx.ShouldBindJSON(&adjustment); err != nil {
		tools.Response(ctx, "CreateAdjustment", false, "Carga útil de solicitud no válida", "create_adjustment", nil, false, "", false)
		return
	}

	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	response := c.Service.CreateAdjustment(userId, adjustment)
	if response != nil {
		tools.Response(ctx, "CreateAdjustment", false, response.Message, "create_adjustment", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateAdjustment", true, "Ajuste creado con éxito", "create_adjustment", adjustment, false, "", false)
}

func (c *AdjustmentsController) ExportAdjustmentsToExcel(ctx *gin.Context) {
	data, response := c.Service.ExportAdjustmentsToExcel()
	if response != nil {
		tools.Response(ctx, "ExportAdjustmentsToExcel", false, response.Message, "export_adjustments_to_excel", nil, false, "", response.Handled)
		return
	}

	if data == nil {
		tools.Response(ctx, "ExportAdjustmentsToExcel", true, "No hay ajustes para exportar", "export_adjustments_to_excel", nil, false, "", true)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=adjustments.xlsx")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
