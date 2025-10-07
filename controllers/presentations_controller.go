package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type PresentationsController struct {
	Service services.PresentationsService
}

func NewPresentationsController(service services.PresentationsService) *PresentationsController {
	return &PresentationsController{
		Service: service,
	}
}

func (c *PresentationsController) GetAllPresentations(ctx *gin.Context) {
	presentations, response := c.Service.GetAllPresentations()

	if response != nil {
		tools.Response(ctx, "GetAllPresentations", false, response.Message, "get_all_presentations", nil, false, "", response.Handled)
		return
	}

	if len(presentations) == 0 {
		tools.Response(ctx, "GetAllPresentations", true, "No se encontraron presentaciones", "get_all_presentations", nil, false, "", true)
		return
	}

	tools.Response(ctx, "GetAllPresentations", true, "Presentaciones recuperadas con éxito", "get_all_presentations", presentations, false, "", false)
}

func (c *PresentationsController) GetPresentationByID(ctx *gin.Context) {
	id := ctx.Param("id")

	presentation, response := c.Service.GetPresentationByID(id)

	if response != nil {
		tools.Response(ctx, "GetPresentationByID", false, response.Message, "get_presentation_by_id", nil, false, "", response.Handled)
		return
	}

	if presentation == nil {
		tools.Response(ctx, "GetPresentationByID", false, "Presentación no encontrada", "get_presentation_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetPresentationByID", true, "Presentación recuperada con éxito", "get_presentation_by_id", presentation, false, "", false)
}

func (c *PresentationsController) CreatePresentation(ctx *gin.Context) {
	var presentation database.Presentations

	if err := ctx.ShouldBindJSON(&presentation); err != nil {
		tools.Response(ctx, "CreatePresentation", false, "Cuerpo de solicitud no válido", "create_presentation", nil, false, "", true)
		return
	}

	response := c.Service.CreatePresentation(&presentation)

	if response != nil {
		tools.Response(ctx, "CreatePresentation", false, response.Message, "create_presentation", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreatePresentation", true, "Presentación creada con éxito", "create_presentation", presentation, false, "", false)
}

func (c *PresentationsController) UpdatePresentation(ctx *gin.Context) {
	id := ctx.Param("id")

	var reqBody struct {
		Description string `json:"description" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		tools.Response(ctx, "UpdatePresentation", false, "Cuerpo de solicitud no válido", "update_presentation", nil, false, "", true)
		return
	}

	presentation, response := c.Service.UpdatePresentation(id, reqBody.Description)

	if response != nil {
		tools.Response(ctx, "UpdatePresentation", false, response.Message, "update_presentation", nil, false, "", response.Handled)
		return
	}

	if presentation == nil {
		tools.Response(ctx, "UpdatePresentation", false, "Presentación no encontrada", "update_presentation", nil, false, "", false)
		return
	}

	tools.Response(ctx, "UpdatePresentation", true, "Presentación actualizada con éxito", "update_presentation", presentation, false, "", false)
}

func (c *PresentationsController) DeletePresentation(ctx *gin.Context) {
	id := ctx.Param("id")

	response := c.Service.DeletePresentation(id)

	if response != nil {
		tools.Response(ctx, "DeletePresentation", false, response.Message, "delete_presentation", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeletePresentation", true, "Presentación eliminada con éxito", "delete_presentation", nil, false, "", false)
}
