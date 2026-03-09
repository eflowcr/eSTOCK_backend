package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type LocationTypesController struct {
	Service services.LocationTypesService
}

func NewLocationTypesController(service services.LocationTypesService) *LocationTypesController {
	return &LocationTypesController{Service: service}
}

func (c *LocationTypesController) ListLocationTypes(ctx *gin.Context) {
	list, resp := c.Service.ListLocationTypes()
	if resp != nil {
		writeErrorResponse(ctx, "ListLocationTypes", "list_location_types", resp)
		return
	}
	tools.ResponseOK(ctx, "ListLocationTypes", "Location types retrieved", "list_location_types", list, false, "")
}

func (c *LocationTypesController) ListLocationTypesAdmin(ctx *gin.Context) {
	list, resp := c.Service.ListLocationTypesAdmin()
	if resp != nil {
		writeErrorResponse(ctx, "ListLocationTypesAdmin", "list_location_types_admin", resp)
		return
	}
	tools.ResponseOK(ctx, "ListLocationTypesAdmin", "Location types (admin) retrieved", "list_location_types_admin", list, false, "")
}

func (c *LocationTypesController) GetLocationTypeByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetLocationTypeByID", "get_location_type_by_id", "Invalid location type ID")
	if !ok {
		return
	}
	lt, resp := c.Service.GetLocationTypeByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetLocationTypeByID", "get_location_type_by_id", resp)
		return
	}
	if lt == nil {
		tools.ResponseNotFound(ctx, "GetLocationTypeByID", "Location type not found", "get_location_type_by_id")
		return
	}
	tools.ResponseOK(ctx, "GetLocationTypeByID", "Location type retrieved", "get_location_type_by_id", lt, false, "")
}

func (c *LocationTypesController) CreateLocationType(ctx *gin.Context) {
	var body requests.LocationTypeCreate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateLocationType", "Invalid request body", "create_location_type")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateLocationType", "create_location_type", errs)
		return
	}
	created, resp := c.Service.CreateLocationType(&body)
	if resp != nil {
		writeErrorResponse(ctx, "CreateLocationType", "create_location_type", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateLocationType", "Location type created", "create_location_type", created, false, "")
}

func (c *LocationTypesController) UpdateLocationType(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateLocationType", "update_location_type", "Invalid location type ID")
	if !ok {
		return
	}
	var body requests.LocationTypeUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateLocationType", "Invalid request body", "update_location_type")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateLocationType", "update_location_type", errs)
		return
	}
	updated, resp := c.Service.UpdateLocationType(id, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateLocationType", "update_location_type", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateLocationType", "Location type updated", "update_location_type", updated, false, "")
}

func (c *LocationTypesController) DeleteLocationType(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteLocationType", "delete_location_type", "Invalid location type ID")
	if !ok {
		return
	}
	resp := c.Service.DeleteLocationType(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeleteLocationType", "delete_location_type", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteLocationType", "Location type deleted", "delete_location_type", nil, false, "")
}
