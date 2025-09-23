package controllers

import (
	"io"
	"strconv"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
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
		tools.Response(ctx, "GetAllArticles", false, response.Message, "get_all_articles", nil, false, "", response.Handled)
		return
	}

	if len(articles) == 0 {
		tools.Response(ctx, "GetAllArticles", true, "No articles found", "get_all_articles", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllArticles", true, "Articles retrieved successfully", "get_all_articles", articles, false, "", false)
}

func (c *ArticlesController) GetArticleByID(ctx *gin.Context) {
	id := ctx.Param("id")

	// Convert id to int
	articleID, err := strconv.Atoi(id)

	if err != nil {
		tools.Response(ctx, "GetArticleByID", false, "Invalid article ID", "get_article_by_id", nil, false, "", false)
		return
	}

	article, response := c.Service.GetArticleByID(articleID)

	if response != nil {
		tools.Response(ctx, "GetArticleByID", false, response.Message, "get_article_by_id", nil, false, "", response.Handled)
		return
	}

	if article == nil {
		tools.Response(ctx, "GetArticleByID", false, "Article not found", "get_article_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetArticleByID", true, "Article retrieved successfully", "get_article_by_id", article, false, "", false)
}

func (c *ArticlesController) GetBySku(ctx *gin.Context) {
	sku := ctx.Param("sku")
	article, response := c.Service.GetBySku(sku)

	if response != nil {
		tools.Response(ctx, "GetBySku", false, response.Message, "get_by_sku", nil, false, "", response.Handled)
		return
	}

	if article == nil {
		tools.Response(ctx, "GetBySku", false, "Article not found", "get_by_sku", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetBySku", true, "Article retrieved successfully", "get_by_sku", article, false, "", false)
}

func (c *ArticlesController) CreateArticle(ctx *gin.Context) {
	var articleRequest requests.Article
	if err := ctx.ShouldBindJSON(&articleRequest); err != nil {
		tools.Response(ctx, "CreateArticle", false, "Invalid request data", "create_article", nil, false, "", false)
		return
	}

	response := c.Service.CreateArticle(&articleRequest)

	if response != nil {
		tools.Response(ctx, "CreateArticle", false, response.Message, "create_article", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateArticle", true, "Article created successfully", "create_article", nil, false, "", false)
}

func (c *ArticlesController) UpdateArticle(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		tools.Response(ctx, "UpdateArticle", false, "Invalid ID", "update_article", nil, false, "", false)
		return
	}

	var req requests.Article
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.Response(ctx, "UpdateArticle", false, "Invalid input: "+err.Error(), "update_article", nil, false, "", false)
		return
	}

	updatedArticle, errResp, warnings := c.Service.UpdateArticle(id, &req)
	if errResp != nil {
		tools.Response(ctx, "UpdateArticle", false, errResp.Message, "update_article", nil, errResp.Handled, "", errResp.Handled)
		return
	}

	response := gin.H{
		"article": updatedArticle,
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	tools.Response(ctx, "UpdateArticle", true, "Article updated successfully", "update_article", response, false, "", false)
}

func (c *ArticlesController) ImportArticlesFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportArticlesFromExcel", false, "File upload error: "+err.Error(), "import_articles_from_excel", nil, false, "", false)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportArticlesFromExcel", false, "Failed to open file: "+err.Error(), "import_articles_from_excel", nil, false, "", false)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportArticlesFromExcel", false, "Failed to read file content: "+err.Error(), "import_articles_from_excel", nil, false, "", false)
		return
	}

	importedArticles, errorResponses := c.Service.ImportArticlesFromExcel(fileBytes)

	if len(importedArticles) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		tools.Response(ctx, "ImportArticlesFromExcel", false, resp.Message, "import_articles_from_excel", nil, false, "", false)
		return
	}

	tools.Response(ctx, "ImportArticlesFromExcel", true, "Articles imported successfully", "import_articles_from_excel", gin.H{
		"imported_articles": importedArticles,
		"errors":            errorResponses,
	}, false, "", false)
}

func (c *ArticlesController) ExportArticlesToExcel(ctx *gin.Context) {
	fileBytes, response := c.Service.ExportArticlesToExcel()
	if response != nil {
		tools.Response(ctx, "ExportArticlesToExcel", false, response.Message, "export_articles_to_excel", nil, false, "", response.Handled)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="articles.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

func (c *ArticlesController) DeleteArticle(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		tools.Response(ctx, "DeleteArticle", false, "Invalid article ID", "delete_article", nil, false, "", false)
		return
	}

	resp := c.Service.DeleteArticle(id)
	if resp != nil {
		tools.Response(ctx, "DeleteArticle", false, resp.Message, "delete_article", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "DeleteArticle", true, "Article deleted successfully", "delete_article", nil, false, "", false)
}
