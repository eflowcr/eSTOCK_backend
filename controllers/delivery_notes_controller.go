package controllers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// DeliveryNotesController handles HTTP for delivery note endpoints (DN3 list + download).
type DeliveryNotesController struct {
	Service  *services.DeliveryNotesService
	TenantID string
}

func NewDeliveryNotesController(svc *services.DeliveryNotesService, tenantID string) *DeliveryNotesController {
	return &DeliveryNotesController{Service: svc, TenantID: tenantID}
}

// List handles GET /api/delivery-notes/
func (c *DeliveryNotesController) List(ctx *gin.Context) {
	var customerID, soNumber, from, to *string

	if v := ctx.Query("customer_id"); v != "" {
		customerID = &v
	}
	if v := ctx.Query("so_number"); v != "" {
		soNumber = &v
	}
	if v := ctx.Query("from"); v != "" {
		from = &v
	}
	if v := ctx.Query("to"); v != "" {
		to = &v
	}

	page := 1
	limit := 50
	if p := ctx.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	result, resp := c.Service.List(c.resolveTenantID(ctx), customerID, soNumber, from, to, page, limit)
	if resp != nil {
		writeErrorResponse(ctx, "ListDeliveryNotes", "list_delivery_notes", resp)
		return
	}
	tools.ResponseOK(ctx, "ListDeliveryNotes", "Notas de entrega recuperadas", "list_delivery_notes", result, false, "")
}

// GetByID handles GET /api/delivery-notes/:id
func (c *DeliveryNotesController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetDeliveryNote", "get_delivery_note", "ID de nota de entrega inválido")
	if !ok {
		return
	}

	dn, resp := c.Service.GetByID(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "GetDeliveryNote", "get_delivery_note", resp)
		return
	}
	tools.ResponseOK(ctx, "GetDeliveryNote", "Nota de entrega recuperada", "get_delivery_note", dn, false, "")
}

// DownloadPDF handles GET /api/delivery-notes/:id/pdf
// Streams the PDF from local FS. Returns 202 if PDF not yet generated.
func (c *DeliveryNotesController) DownloadPDF(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DownloadDNPDF", "download_dn_pdf", "ID de nota de entrega inválido")
	if !ok {
		return
	}

	// Verify DN exists and belongs to tenant.
	dnNumber, resp := c.Service.GetDNNumber(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "DownloadDNPDF", "download_dn_pdf", resp)
		return
	}

	pdfPath := services.PDFLocalPath(id)
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		// PDF not yet generated — return 202 Accepted.
		ctx.JSON(http.StatusAccepted, gin.H{
			"message": "PDF not yet generated. Please retry shortly.",
			"dn_id":   id,
		})
		return
	}

	filename := dnNumber + ".pdf"
	ctx.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	ctx.Header("Content-Type", "application/pdf")
	ctx.File(pdfPath)
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *DeliveryNotesController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
