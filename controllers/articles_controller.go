package controllers

import (
	"encoding/json"
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// ArticlesController — S3.5 W5.5 (HR-S3.5 C1): tenant is sourced per-request from the
// JWT claim (TenantIDFromContext). The TenantID field is kept ONLY as a fallback for
// system / cron / admin paths that bypass JWTAuthMiddleware and for tests that
// pre-construct the controller with a default tenant.
type ArticlesController struct {
	Service       services.ArticlesService
	AuditService  *services.AuditService
	UserPrefsRepo ports.UserPreferencesRepository
	TenantID      string // fallback for non-JWT callers only
}

func NewArticlesController(service services.ArticlesService, auditSvc *services.AuditService, userPrefsRepo ports.UserPreferencesRepository, tenantID string) *ArticlesController {
	return &ArticlesController{
		Service:       service,
		AuditService:  auditSvc,
		UserPrefsRepo: userPrefsRepo,
		TenantID:      tenantID,
	}
}

// resolveTenantID returns the JWT tenant claim, falling back to the env-injected default.
// Returns "" iff neither is set — the caller MUST then 401 to avoid cross-tenant leaks.
func (c *ArticlesController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}

func (c *ArticlesController) GetAllArticles(ctx *gin.Context) {
	articles, response := c.Service.GetAllArticles(c.resolveTenantID(ctx))

	if response != nil {
		writeErrorResponse(ctx, "GetAllArticles", "get_all_articles", response)
		return
	}

	if len(articles) == 0 {
		tools.ResponseOK(ctx, "GetAllArticles", "No se encontraron artículos", "get_all_articles", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllArticles", "Artículos recuperados con éxito", "get_all_articles", articles, false, "")
}

func (c *ArticlesController) GetArticleByID(ctx *gin.Context) {
	articleID, ok := tools.ParseRequiredParam(ctx, "id", "GetArticleByID", "get_article_by_id", "ID de artículo no válido")
	if !ok {
		return
	}

	article, response := c.Service.GetArticleByID(articleID, c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "GetArticleByID", "get_article_by_id", response)
		return
	}

	if article == nil {
		tools.ResponseNotFound(ctx, "GetArticleByID", "Artículo no encontrado", "get_article_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetArticleByID", "Artículo recuperado con éxito", "get_article_by_id", c.Service.EnrichArticle(article), false, "")
}

func (c *ArticlesController) GetBySku(ctx *gin.Context) {
	sku, ok := tools.ParseRequiredParam(ctx, "sku", "GetBySku", "get_by_sku", "SKU inválido")
	if !ok {
		return
	}
	article, response := c.Service.GetBySku(sku, c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "GetBySku", "get_by_sku", response)
		return
	}

	if article == nil {
		tools.ResponseNotFound(ctx, "GetBySku", "Artículo no encontrado", "get_by_sku")
		return
	}

	tools.ResponseOK(ctx, "GetBySku", "Artículo recuperado con éxito", "get_by_sku", article, false, "")
}

func (c *ArticlesController) CreateArticle(ctx *gin.Context) {
	var articleRequest requests.Article
	if err := ctx.ShouldBindJSON(&articleRequest); err != nil {
		tools.ResponseBadRequest(ctx, "CreateArticle", "Datos de solicitud inválidos", "create_article")
		return
	}
	if errs := tools.ValidateStruct(&articleRequest); errs != nil {
		tools.ResponseValidationError(ctx, "CreateArticle", "create_article", errs)
		return
	}

	response := c.Service.CreateArticle(c.resolveTenantID(ctx), &articleRequest)
	if response != nil {
		writeErrorResponse(ctx, "CreateArticle", "create_article", response)
		return
	}

	if c.AuditService != nil {
		userIDVal, _ := ctx.Get(tools.ContextKeyUserID)
		var uid *string
		if idStr, ok := userIDVal.(string); ok && idStr != "" {
			uid = &idStr
		}
		newVal, _ := json.Marshal(articleRequest)
		c.AuditService.Log(ctx.Request.Context(), uid, tools.ActionCreate, tools.ResourceArticle, "", nil, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseCreated(ctx, "CreateArticle", "Artículo creado con éxito", "create_article", nil, false, "")
}

func (c *ArticlesController) UpdateArticle(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateArticle", "update_article", "ID de artículo no válido")
	if !ok {
		return
	}

	article, _ := c.Service.GetArticleByID(id, c.resolveTenantID(ctx))
	var req requests.Article
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateArticle", "Carga útil no válida", "update_article")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateArticle", "update_article", errs)
		return
	}

	updatedArticle, errResp, warnings := c.Service.UpdateArticle(id, c.resolveTenantID(ctx), &req)
	if errResp != nil {
		writeErrorResponse(ctx, "UpdateArticle", "update_article", errResp)
		return
	}

	if c.AuditService != nil && updatedArticle != nil {
		userIDVal, _ := ctx.Get(tools.ContextKeyUserID)
		var uid *string
		if idStr, ok := userIDVal.(string); ok && idStr != "" {
			uid = &idStr
		}
		oldVal, _ := json.Marshal(article)
		newVal, _ := json.Marshal(updatedArticle)
		c.AuditService.Log(ctx.Request.Context(), uid, tools.ActionUpdate, tools.ResourceArticle, id, oldVal, newVal, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	payload := gin.H{"article": c.Service.EnrichArticle(updatedArticle)}
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	tools.ResponseOK(ctx, "UpdateArticle", "Artículo actualizado con éxito", "update_article", payload, false, "")
}

func (c *ArticlesController) ImportArticlesFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportArticlesFromExcel", "Error al subir el archivo", "import_articles_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportArticlesFromExcel", "Error al abrir el archivo", "import_articles_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportArticlesFromExcel", "Error al leer el contenido del archivo", "import_articles_from_excel")
		return
	}

	importedArticles, skippedArticles, errorResponses := c.Service.ImportArticlesFromExcel(c.resolveTenantID(ctx), fileBytes)
	if len(importedArticles) == 0 && len(skippedArticles) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		writeErrorResponse(ctx, "ImportArticlesFromExcel", "import_articles_from_excel", resp)
		return
	}

	tools.ResponseOK(ctx, "ImportArticlesFromExcel", "Artículos importados con éxito", "import_articles_from_excel", gin.H{
		"successful":   len(importedArticles),
		"skipped":      len(skippedArticles),
		"failed":       len(errorResponses),
		"imported":     importedArticles,
		"skipped_rows": skippedArticles,
		"errors":       errorResponses,
	}, false, "")
}

func (c *ArticlesController) ValidateImportRows(ctx *gin.Context) {
	var rows []requests.ArticleImportRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		tools.ResponseBadRequest(ctx, "ValidateImportRows", "JSON inválido", "validate_import_rows")
		return
	}
	if len(rows) == 0 {
		tools.ResponseBadRequest(ctx, "ValidateImportRows", "No se proporcionaron filas", "validate_import_rows")
		return
	}
	results, resp := c.Service.ValidateImportRows(c.resolveTenantID(ctx), rows)
	if resp != nil {
		writeErrorResponse(ctx, "ValidateImportRows", "validate_import_rows", resp)
		return
	}
	tools.ResponseOK(ctx, "ValidateImportRows", "Validación completada", "validate_import_rows", gin.H{
		"results": results,
	}, false, "")
}

func (c *ArticlesController) ImportArticlesFromJSON(ctx *gin.Context) {
	var rows []requests.ArticleImportRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		tools.ResponseBadRequest(ctx, "ImportArticlesFromJSON", "JSON inválido", "import_articles_from_json")
		return
	}
	if len(rows) == 0 {
		tools.ResponseBadRequest(ctx, "ImportArticlesFromJSON", "No se proporcionaron filas para importar", "import_articles_from_json")
		return
	}

	imported, skipped, errorResponses := c.Service.ImportArticlesFromJSON(c.resolveTenantID(ctx), rows)
	tools.ResponseOK(ctx, "ImportArticlesFromJSON", "Importación completada", "import_articles_from_json", gin.H{
		"successful":   len(imported),
		"skipped":      len(skipped),
		"failed":       len(errorResponses),
		"imported":     imported,
		"skipped_rows": skipped,
		"errors":       errorResponses,
	}, false, "")
}

func (c *ArticlesController) ExportArticlesToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportArticlesToExcel(c.resolveTenantID(ctx))
	if response != nil {
		writeErrorResponse(ctx, "ExportArticlesToExcel", "export_articles_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="articles.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ArticlesController) DownloadImportTemplate(ctx *gin.Context) {
	lang := "es"
	uid := ctx.GetString(tools.ContextKeyUserID)
	if uid != "" && c.UserPrefsRepo != nil {
		if prefs, err := c.UserPrefsRepo.GetOrCreateUserPreferences(ctx.Request.Context(), uid); err == nil && prefs != nil && prefs.Language != "" {
			lang = prefs.Language
		}
	}

	fileBytes, response := c.Service.GenerateImportTemplate(c.resolveTenantID(ctx), lang)
	if response != nil {
		writeErrorResponse(ctx, "DownloadImportTemplate", "download_articles_import_template", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="ImportArticles.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ArticlesController) DeleteArticle(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteArticle", "delete_article", "ID de artículo no válido")
	if !ok {
		return
	}

	article, _ := c.Service.GetArticleByID(id, c.resolveTenantID(ctx))
	resp := c.Service.DeleteArticle(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "DeleteArticle", "delete_article", resp)
		return
	}

	if c.AuditService != nil && article != nil {
		userID, _ := ctx.Get(tools.ContextKeyUserID)
		var uid *string
		if idStr, ok := userID.(string); ok && idStr != "" {
			uid = &idStr
		}
		oldVal, _ := json.Marshal(article)
		c.AuditService.Log(ctx.Request.Context(), uid, tools.ActionDelete, tools.ResourceArticle, id, oldVal, nil, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
	tools.ResponseOK(ctx, "DeleteArticle", "Artículo eliminado con éxito", "delete_article", nil, false, "")
}

// writeErrorResponse sends the appropriate HTTP status based on response.StatusCode, or legacy 200/400 from Handled.
func writeErrorResponse(ctx *gin.Context, transactionType, endpointCode string, resp *responses.InternalResponse) {
	switch resp.StatusCode {
	case responses.StatusBadRequest:
		tools.ResponseBadRequest(ctx, transactionType, resp.Message, endpointCode)
	case responses.StatusForbidden:
		tools.ResponseForbidden(ctx, transactionType, resp.Message, endpointCode)
	case responses.StatusNotFound:
		tools.ResponseNotFound(ctx, transactionType, resp.Message, endpointCode)
	case responses.StatusConflict:
		tools.ResponseConflict(ctx, transactionType, resp.Message, endpointCode)
	case responses.StatusInternalServerError:
		tools.ResponseInternal(ctx, transactionType, resp.Message, endpointCode)
	default:
		tools.Response(ctx, transactionType, false, resp.Message, endpointCode, nil, false, "", resp.Handled)
	}
}
