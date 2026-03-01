package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type ArticlesController struct {
	Service services.ArticlesService
}

func NewArticlesController(service services.ArticlesService) *ArticlesController {
	return &ArticlesController{
		Service: service,
	}
}

func (c *ArticlesController) GetAllArticles(ctx *gin.Context) {
	articles, response := c.Service.GetAllArticles()

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
	articleID, ok := tools.ParseIntParam(ctx, "id", "GetArticleByID", "get_article_by_id", "ID de artículo no válido")
	if !ok {
		return
	}

	article, response := c.Service.GetArticleByID(articleID)
	if response != nil {
		writeErrorResponse(ctx, "GetArticleByID", "get_article_by_id", response)
		return
	}

	if article == nil {
		tools.ResponseNotFound(ctx, "GetArticleByID", "Artículo no encontrado", "get_article_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetArticleByID", "Artículo recuperado con éxito", "get_article_by_id", article, false, "")
}

func (c *ArticlesController) GetBySku(ctx *gin.Context) {
	sku := ctx.Param("sku")
	article, response := c.Service.GetBySku(sku)
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

	response := c.Service.CreateArticle(&articleRequest)
	if response != nil {
		writeErrorResponse(ctx, "CreateArticle", "create_article", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateArticle", "Artículo creado con éxito", "create_article", nil, false, "")
}

func (c *ArticlesController) UpdateArticle(ctx *gin.Context) {
	id, ok := tools.ParseIntParam(ctx, "id", "UpdateArticle", "update_article", "ID de artículo no válido")
	if !ok {
		return
	}

	var req requests.Article
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateArticle", "Carga útil no válida", "update_article")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateArticle", "update_article", errs)
		return
	}

	updatedArticle, errResp, warnings := c.Service.UpdateArticle(id, &req)
	if errResp != nil {
		writeErrorResponse(ctx, "UpdateArticle", "update_article", errResp)
		return
	}

	payload := gin.H{"article": updatedArticle}
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

	importedArticles, errorResponses := c.Service.ImportArticlesFromExcel(fileBytes)
	if len(importedArticles) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		writeErrorResponse(ctx, "ImportArticlesFromExcel", "import_articles_from_excel", resp)
		return
	}

	tools.ResponseOK(ctx, "ImportArticlesFromExcel", "Artículos importados con éxito", "import_articles_from_excel", gin.H{
		"imported_articles": importedArticles,
		"errors":            errorResponses,
	}, false, "")
}

func (c *ArticlesController) ExportArticlesToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportArticlesToExcel()
	if response != nil {
		writeErrorResponse(ctx, "ExportArticlesToExcel", "export_articles_to_excel", response)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="articles.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ArticlesController) DeleteArticle(ctx *gin.Context) {
	id, ok := tools.ParseIntParam(ctx, "id", "DeleteArticle", "delete_article", "ID de artículo no válido")
	if !ok {
		return
	}

	resp := c.Service.DeleteArticle(id)
	if resp != nil {
		writeErrorResponse(ctx, "DeleteArticle", "delete_article", resp)
		return
	}

	tools.ResponseOK(ctx, "DeleteArticle", "Artículo eliminado con éxito", "delete_article", nil, false, "")
}

// writeErrorResponse sends the appropriate HTTP status based on response.StatusCode, or legacy 200/400 from Handled.
func writeErrorResponse(ctx *gin.Context, transactionType, endpointCode string, resp *responses.InternalResponse) {
	switch resp.StatusCode {
	case responses.StatusBadRequest:
		tools.ResponseBadRequest(ctx, transactionType, resp.Message, endpointCode)
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
