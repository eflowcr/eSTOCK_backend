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
		writeErrorResponse(ctx, "GetAllPresentations", "get_all_presentations", response)
		return
	}

	if len(presentations) == 0 {
		tools.ResponseOK(ctx, "GetAllPresentations", "No se encontraron presentaciones", "get_all_presentations", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllPresentations", "Presentaciones recuperadas con éxito", "get_all_presentations", presentations, false, "")
}

func (c *PresentationsController) GetPresentationByID(ctx *gin.Context) {
	id := ctx.Param("id")

	presentation, response := c.Service.GetPresentationByID(id)

	if response != nil {
		writeErrorResponse(ctx, "GetPresentationByID", "get_presentation_by_id", response)
		return
	}

	if presentation == nil {
		tools.ResponseNotFound(ctx, "GetPresentationByID", "Presentación no encontrada", "get_presentation_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetPresentationByID", "Presentación recuperada con éxito", "get_presentation_by_id", presentation, false, "")
}

func (c *PresentationsController) CreatePresentation(ctx *gin.Context) {
	var presentation database.Presentations

	if err := ctx.ShouldBindJSON(&presentation); err != nil {
		tools.ResponseBadRequest(ctx, "CreatePresentation", "Cuerpo de solicitud no válido", "create_presentation")
		return
	}
	if errs := tools.ValidateStruct(&presentation); errs != nil {
		tools.ResponseValidationError(ctx, "CreatePresentation", "create_presentation", errs)
		return
	}

	response := c.Service.CreatePresentation(&presentation)

	if response != nil {
		writeErrorResponse(ctx, "CreatePresentation", "create_presentation", response)
		return
	}

	tools.ResponseCreated(ctx, "CreatePresentation", "Presentación creada con éxito", "create_presentation", presentation, false, "")
}

func (c *PresentationsController) UpdatePresentation(ctx *gin.Context) {
	id := ctx.Param("id")

	var reqBody struct {
		Description string `json:"description" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePresentation", "Cuerpo de solicitud no válido", "update_presentation")
		return
	}
	if errs := tools.ValidateStruct(&reqBody); errs != nil {
		tools.ResponseValidationError(ctx, "UpdatePresentation", "update_presentation", errs)
		return
	}

	presentation, response := c.Service.UpdatePresentation(id, reqBody.Description)

	if response != nil {
		writeErrorResponse(ctx, "UpdatePresentation", "update_presentation", response)
		return
	}

	if presentation == nil {
		tools.ResponseNotFound(ctx, "UpdatePresentation", "Presentación no encontrada", "update_presentation")
		return
	}

	tools.ResponseOK(ctx, "UpdatePresentation", "Presentación actualizada con éxito", "update_presentation", presentation, false, "")
}

func (c *PresentationsController) DeletePresentation(ctx *gin.Context) {
	id := ctx.Param("id")

	response := c.Service.DeletePresentation(id)

	if response != nil {
		writeErrorResponse(ctx, "DeletePresentation", "delete_presentation", response)
		return
	}

	tools.ResponseOK(ctx, "DeletePresentation", "Presentación eliminada con éxito", "delete_presentation", nil, false, "")
}
