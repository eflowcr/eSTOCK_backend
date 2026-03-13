package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type PresentationTypesController struct {
	Service services.PresentationTypesService
}

func NewPresentationTypesController(service services.PresentationTypesService) *PresentationTypesController {
	return &PresentationTypesController{Service: service}
}

func (c *PresentationTypesController) ListPresentationTypes(ctx *gin.Context) {
	list, resp := c.Service.ListPresentationTypes()
	if resp != nil {
		writeErrorResponse(ctx, "ListPresentationTypes", "list_presentation_types", resp)
		return
	}
	tools.ResponseOK(ctx, "ListPresentationTypes", "Presentation types retrieved", "list_presentation_types", list, false, "")
}

func (c *PresentationTypesController) ListPresentationTypesAdmin(ctx *gin.Context) {
	list, resp := c.Service.ListPresentationTypesAdmin()
	if resp != nil {
		writeErrorResponse(ctx, "ListPresentationTypesAdmin", "list_presentation_types_admin", resp)
		return
	}
	tools.ResponseOK(ctx, "ListPresentationTypesAdmin", "Presentation types (admin) retrieved", "list_presentation_types_admin", list, false, "")
}

func (c *PresentationTypesController) GetPresentationTypeByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetPresentationTypeByID", "get_presentation_type_by_id", "Invalid presentation type ID")
	if !ok {
		return
	}
	pt, resp := c.Service.GetPresentationTypeByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetPresentationTypeByID", "get_presentation_type_by_id", resp)
		return
	}
	if pt == nil {
		tools.ResponseNotFound(ctx, "GetPresentationTypeByID", "Presentation type not found", "get_presentation_type_by_id")
		return
	}
	tools.ResponseOK(ctx, "GetPresentationTypeByID", "Presentation type retrieved", "get_presentation_type_by_id", pt, false, "")
}

func (c *PresentationTypesController) CreatePresentationType(ctx *gin.Context) {
	var body requests.PresentationTypeCreate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreatePresentationType", "Invalid request body", "create_presentation_type")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreatePresentationType", "create_presentation_type", errs)
		return
	}
	created, resp := c.Service.CreatePresentationType(&body)
	if resp != nil {
		writeErrorResponse(ctx, "CreatePresentationType", "create_presentation_type", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreatePresentationType", "Presentation type created", "create_presentation_type", created, false, "")
}

func (c *PresentationTypesController) UpdatePresentationType(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdatePresentationType", "update_presentation_type", "Invalid presentation type ID")
	if !ok {
		return
	}
	var body requests.PresentationTypeUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePresentationType", "Invalid request body", "update_presentation_type")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "UpdatePresentationType", "update_presentation_type", errs)
		return
	}
	updated, resp := c.Service.UpdatePresentationType(id, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdatePresentationType", "update_presentation_type", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdatePresentationType", "Presentation type updated", "update_presentation_type", updated, false, "")
}

func (c *PresentationTypesController) DeletePresentationType(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeletePresentationType", "delete_presentation_type", "Invalid presentation type ID")
	if !ok {
		return
	}
	resp := c.Service.DeletePresentationType(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeletePresentationType", "delete_presentation_type", resp)
		return
	}
	tools.ResponseOK(ctx, "DeletePresentationType", "Presentation type deleted", "delete_presentation_type", nil, false, "")
}
