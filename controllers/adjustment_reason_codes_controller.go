package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type AdjustmentReasonCodesController struct {
	Service services.AdjustmentReasonCodesService
}

func NewAdjustmentReasonCodesController(service services.AdjustmentReasonCodesService) *AdjustmentReasonCodesController {
	return &AdjustmentReasonCodesController{Service: service}
}

func (c *AdjustmentReasonCodesController) ListAdjustmentReasonCodes(ctx *gin.Context) {
	list, resp := c.Service.ListAdjustmentReasonCodes()
	if resp != nil {
		writeErrorResponse(ctx, "ListAdjustmentReasonCodes", "list_adjustment_reason_codes", resp)
		return
	}
	tools.ResponseOK(ctx, "ListAdjustmentReasonCodes", "Adjustment reason codes retrieved", "list_adjustment_reason_codes", list, false, "")
}

func (c *AdjustmentReasonCodesController) ListAdjustmentReasonCodesAdmin(ctx *gin.Context) {
	list, resp := c.Service.ListAdjustmentReasonCodesAdmin()
	if resp != nil {
		writeErrorResponse(ctx, "ListAdjustmentReasonCodesAdmin", "list_adjustment_reason_codes_admin", resp)
		return
	}
	tools.ResponseOK(ctx, "ListAdjustmentReasonCodesAdmin", "Adjustment reason codes (admin) retrieved", "list_adjustment_reason_codes_admin", list, false, "")
}

func (c *AdjustmentReasonCodesController) GetAdjustmentReasonCodeByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetAdjustmentReasonCodeByID", "get_adjustment_reason_code_by_id", "Invalid adjustment reason code ID")
	if !ok {
		return
	}
	rc, resp := c.Service.GetAdjustmentReasonCodeByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetAdjustmentReasonCodeByID", "get_adjustment_reason_code_by_id", resp)
		return
	}
	if rc == nil {
		tools.ResponseNotFound(ctx, "GetAdjustmentReasonCodeByID", "Adjustment reason code not found", "get_adjustment_reason_code_by_id")
		return
	}
	tools.ResponseOK(ctx, "GetAdjustmentReasonCodeByID", "Adjustment reason code retrieved", "get_adjustment_reason_code_by_id", rc, false, "")
}

func (c *AdjustmentReasonCodesController) CreateAdjustmentReasonCode(ctx *gin.Context) {
	var body requests.AdjustmentReasonCodeCreate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "CreateAdjustmentReasonCode", "Invalid request body", "create_adjustment_reason_code")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "CreateAdjustmentReasonCode", "create_adjustment_reason_code", errs)
		return
	}
	created, resp := c.Service.CreateAdjustmentReasonCode(&body)
	if resp != nil {
		writeErrorResponse(ctx, "CreateAdjustmentReasonCode", "create_adjustment_reason_code", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateAdjustmentReasonCode", "Adjustment reason code created", "create_adjustment_reason_code", created, false, "")
}

func (c *AdjustmentReasonCodesController) UpdateAdjustmentReasonCode(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateAdjustmentReasonCode", "update_adjustment_reason_code", "Invalid adjustment reason code ID")
	if !ok {
		return
	}
	var body requests.AdjustmentReasonCodeUpdate
	if err := ctx.ShouldBindJSON(&body); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateAdjustmentReasonCode", "Invalid request body", "update_adjustment_reason_code")
		return
	}
	if errs := tools.ValidateStruct(&body); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateAdjustmentReasonCode", "update_adjustment_reason_code", errs)
		return
	}
	updated, resp := c.Service.UpdateAdjustmentReasonCode(id, &body)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateAdjustmentReasonCode", "update_adjustment_reason_code", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateAdjustmentReasonCode", "Adjustment reason code updated", "update_adjustment_reason_code", updated, false, "")
}

func (c *AdjustmentReasonCodesController) DeleteAdjustmentReasonCode(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteAdjustmentReasonCode", "delete_adjustment_reason_code", "Invalid adjustment reason code ID")
	if !ok {
		return
	}
	resp := c.Service.DeleteAdjustmentReasonCode(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeleteAdjustmentReasonCode", "delete_adjustment_reason_code", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteAdjustmentReasonCode", "Adjustment reason code deleted", "delete_adjustment_reason_code", nil, false, "")
}
