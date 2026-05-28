package controllers

import (
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// ArticleSuppliersController exposes CRUD for the article↔supplier link table at
// /api/article-suppliers/* — closing S3-5 ("Múltiples proveedores por artículo") from the
// 2026-04-29 scope email. TenantID is JWT-first with env fallback (same pattern as
// inventory/clients post-v1.4.2 multi-tenant hardening).
type ArticleSuppliersController struct {
	Service  *services.ArticleSuppliersService
	TenantID string
}

func NewArticleSuppliersController(service *services.ArticleSuppliersService, tenantID string) *ArticleSuppliersController {
	return &ArticleSuppliersController{Service: service, TenantID: tenantID}
}

func (c *ArticleSuppliersController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}

func (c *ArticleSuppliersController) List(ctx *gin.Context) {
	var f ports.ArticleSupplierFilters
	if sku := ctx.Query("article_sku"); sku != "" {
		f.ArticleSKU = &sku
	}
	if sup := ctx.Query("supplier_id"); sup != "" {
		f.SupplierID = &sup
	}
	if pref := ctx.Query("is_preferred"); pref != "" {
		if b, err := strconv.ParseBool(pref); err == nil {
			f.IsPreferred = &b
		}
	}
	rows, resp := c.Service.List(c.resolveTenantID(ctx), f)
	if resp != nil {
		writeErrorResponse(ctx, "ListArticleSuppliers", "list_article_suppliers", resp)
		return
	}
	tools.ResponseOK(ctx, "ListArticleSuppliers", "Proveedores del artículo obtenidos", "list_article_suppliers", rows, false, "")
}

func (c *ArticleSuppliersController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetArticleSupplier", "get_article_supplier", "ID requerido")
	if !ok {
		return
	}
	item, resp := c.Service.GetByID(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "GetArticleSupplier", "get_article_supplier", resp)
		return
	}
	tools.ResponseOK(ctx, "GetArticleSupplier", "Relación artículo-proveedor obtenida", "get_article_supplier", item, false, "")
}

func (c *ArticleSuppliersController) Create(ctx *gin.Context) {
	var req requests.CreateArticleSupplierRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "CreateArticleSupplier", "Datos de solicitud inválidos", "create_article_supplier")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "CreateArticleSupplier", "create_article_supplier", errs)
		return
	}
	item, resp := c.Service.Create(c.resolveTenantID(ctx), &req)
	if resp != nil {
		writeErrorResponse(ctx, "CreateArticleSupplier", "create_article_supplier", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateArticleSupplier", "Proveedor agregado al artículo", "create_article_supplier", item, false, "")
}

func (c *ArticleSuppliersController) Update(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateArticleSupplier", "update_article_supplier", "ID requerido")
	if !ok {
		return
	}
	var req requests.UpdateArticleSupplierRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateArticleSupplier", "Datos de solicitud inválidos", "update_article_supplier")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateArticleSupplier", "update_article_supplier", errs)
		return
	}
	item, resp := c.Service.Update(id, c.resolveTenantID(ctx), &req)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateArticleSupplier", "update_article_supplier", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateArticleSupplier", "Relación actualizada", "update_article_supplier", item, false, "")
}

func (c *ArticleSuppliersController) SoftDelete(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteArticleSupplier", "delete_article_supplier", "ID requerido")
	if !ok {
		return
	}
	if resp := c.Service.SoftDelete(id, c.resolveTenantID(ctx)); resp != nil {
		writeErrorResponse(ctx, "DeleteArticleSupplier", "delete_article_supplier", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteArticleSupplier", "Relación eliminada", "delete_article_supplier", nil, false, "")
}
