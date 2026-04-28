package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type CategoriesController struct {
	Service  services.CategoriesService
	TenantID string
}

func NewCategoriesController(service services.CategoriesService, tenantID string) *CategoriesController {
	return &CategoriesController{Service: service, TenantID: tenantID}
}

func (c *CategoriesController) List(ctx *gin.Context) {
	categories, resp := c.Service.ListByTenant(c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "ListCategories", "list_categories", resp)
		return
	}
	tools.ResponseOK(ctx, "ListCategories", "Categorías recuperadas", "list_categories", categories, false, "")
}

func (c *CategoriesController) GetTree(ctx *gin.Context) {
	tree, resp := c.Service.GetTree(c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "GetCategoriesTree", "get_categories_tree", resp)
		return
	}
	tools.ResponseOK(ctx, "GetCategoriesTree", "Árbol de categorías recuperado", "get_categories_tree", tree, false, "")
}

func (c *CategoriesController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetCategoryByID", "get_category", "ID de categoría inválido")
	if !ok {
		return
	}
	cat, resp := c.Service.GetByID(id)
	if resp != nil {
		writeErrorResponse(ctx, "GetCategoryByID", "get_category", resp)
		return
	}
	tools.ResponseOK(ctx, "GetCategoryByID", "Categoría recuperada", "get_category", cat, false, "")
}

func (c *CategoriesController) Create(ctx *gin.Context) {
	var req requests.CreateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "CreateCategory", "Datos de solicitud inválidos", "create_category")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "CreateCategory", "create_category", errs)
		return
	}

	cat, resp := c.Service.Create(c.TenantID, &req)
	if resp != nil {
		writeErrorResponse(ctx, "CreateCategory", "create_category", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateCategory", "Categoría creada exitosamente", "create_category", cat, false, "")
}

func (c *CategoriesController) Update(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateCategory", "update_category", "ID de categoría inválido")
	if !ok {
		return
	}
	var req requests.UpdateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateCategory", "Datos de solicitud inválidos", "update_category")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateCategory", "update_category", errs)
		return
	}

	cat, resp := c.Service.Update(id, &req, c.TenantID)
	if resp != nil {
		writeErrorResponse(ctx, "UpdateCategory", "update_category", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateCategory", "Categoría actualizada", "update_category", cat, false, "")
}

func (c *CategoriesController) SoftDelete(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteCategory", "delete_category", "ID de categoría inválido")
	if !ok {
		return
	}
	if resp := c.Service.SoftDelete(id); resp != nil {
		writeErrorResponse(ctx, "DeleteCategory", "delete_category", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteCategory", "Categoría eliminada", "delete_category", nil, false, "")
}
