package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type AuditController struct {
	Service *services.AuditService
}

func NewAuditController(svc *services.AuditService) *AuditController {
	return &AuditController{Service: svc}
}

// ListAuditLogs handles GET /api/audit-logs with query params: page, per_page, user_id, resource_type, resource_id, action, start_date, end_date.
func (c *AuditController) ListAuditLogs(ctx *gin.Context) {
	if c.Service == nil {
		tools.ResponseInternal(ctx, "ListAuditLogs", "Audit logging no disponible", "list_audit_logs")
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(ctx.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	limit := int32(perPage)
	offset := int32((page - 1) * perPage)

	params := ports.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	}
	if v := ctx.Query("user_id"); v != "" {
		params.FilterUserID = &v
	}
	if v := ctx.Query("resource_type"); v != "" {
		params.FilterResourceType = &v
	}
	if v := ctx.Query("resource_id"); v != "" {
		params.FilterResourceID = &v
	}
	if v := ctx.Query("action"); v != "" {
		params.FilterAction = &v
	}
	if v := ctx.Query("start_date"); v != "" {
		params.FilterStartDate = &v
	}
	if v := ctx.Query("end_date"); v != "" {
		params.FilterEndDate = &v
	}

	entries, total, err := c.Service.List(ctx.Request.Context(), params)
	if err != nil {
		tools.ResponseInternal(ctx, "ListAuditLogs", "Error al obtener registros de auditoria", "list_audit_logs")
		return
	}

	payload := gin.H{
		"data":     entries,
		"page":     page,
		"per_page": perPage,
		"total":    total,
	}
	tools.ResponseOK(ctx, "ListAuditLogs", "Registros de auditoria", "list_audit_logs", payload, false, "")
}
