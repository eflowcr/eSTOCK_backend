package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type PresentationConversionsController struct {
	Service services.PresentationConversionsService
}

func NewPresentationConversionsController(service services.PresentationConversionsService) *PresentationConversionsController {
	return &PresentationConversionsController{Service: service}
}

func (c *PresentationConversionsController) ListPresentationConversions(ctx *gin.Context) {
	list, resp := c.Service.ListPresentationConversions()
	if resp != nil {
		writeErrorResponse(ctx, "ListPresentationConversions", "list_presentation_conversions", resp)
		return
	}
	tools.ResponseOK(ctx, "ListPresentationConversions", "Presentation conversions retrieved", "list_presentation_conversions", list, false, "")
}

func (c *PresentationConversionsController) ListPresentationConversionsAdmin(ctx *gin.Context) {
	list, resp := c.Service.ListPresentationConversionsAdmin()
	if resp != nil {
		writeErrorResponse(ctx, "ListPresentationConversionsAdmin", "list_presentation_conversions_admin", resp)
		return
	}
	tools.ResponseOK(ctx, "ListPresentationConversionsAdmin", "Presentation conversions (admin) retrieved", "list_presentation_conversions_admin", list, false, "")
}

func (c *PresentationConversionsController) GetPresentationConversionByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetPresentationConversionByID", "get_presentation_conversion_by_id", "Invalid presentation conversion ID")
	if !ok {
		return
	}
	pc, resp := c.Service.GetPresentationConversionByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetPresentationConversionByID", "get_presentation_conversion_by_id", resp)
		return
	}
	if pc == nil {
		tools.ResponseNotFound(ctx, "GetPresentationConversionByID", "Presentation conversion not found", "get_presentation_conversion_by_id")
		return
	}
	tools.ResponseOK(ctx, "GetPresentationConversionByID", "Presentation conversion retrieved", "get_presentation_conversion_by_id", pc, false, "")
}

func (c *PresentationConversionsController) CreatePresentationConversion(ctx *gin.Context) {
	var body requests.PresentationConversionCreate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreatePresentationConversion", "Invalid request body", "create_presentation_conversion")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreatePresentationConversion", "create_presentation_conversion", errs)
		return
	}
	created, resp := c.Service.CreatePresentationConversion(&body)
	if resp != nil {
		writeErrorResponse(ctx, "CreatePresentationConversion", "create_presentation_conversion", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreatePresentationConversion", "Presentation conversion created", "create_presentation_conversion", created, false, "")
}

func (c *PresentationConversionsController) UpdatePresentationConversion(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdatePresentationConversion", "update_presentation_conversion", "Invalid presentation conversion ID")
	if !ok {
		return
	}
	var body requests.PresentationConversionUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePresentationConversion", "Invalid request body", "update_presentation_conversion")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "UpdatePresentationConversion", "update_presentation_conversion", errs)
		return
	}
	updated, resp := c.Service.UpdatePresentationConversion(id, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdatePresentationConversion", "update_presentation_conversion", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdatePresentationConversion", "Presentation conversion updated", "update_presentation_conversion", updated, false, "")
}

func (c *PresentationConversionsController) DeletePresentationConversion(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeletePresentationConversion", "delete_presentation_conversion", "Invalid presentation conversion ID")
	if !ok {
		return
	}
	resp := c.Service.DeletePresentationConversion(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeletePresentationConversion", "delete_presentation_conversion", resp)
		return
	}
	tools.ResponseOK(ctx, "DeletePresentationConversion", "Presentation conversion deleted", "delete_presentation_conversion", nil, false, "")
}
