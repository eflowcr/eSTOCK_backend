package repositories

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ArticlesRepository struct {
	DB *gorm.DB
}

func (r *ArticlesRepository) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	var articles []database.Article

	err := r.DB.
		Table(database.Article{}.TableName()).
		Order("created_at ASC").
		Find(&articles).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch articles",
			Handled: false,
		}
	}

	if len(articles) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No articles found",
			Handled: true,
		}
	}

	return articles, nil
}

func (r *ArticlesRepository) GetArticleByID(id int) (*database.Article, *responses.InternalResponse) {
	var article database.Article

	err := r.DB.
		Table(database.Article{}.TableName()).
		Where("id = ?", id).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Article not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article",
			Handled: false,
		}
	}

	return &article, nil
}

func (r *ArticlesRepository) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	var article database.Article

	err := r.DB.
		Table(database.Article{}.TableName()).
		Where("sku = ?", sku).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Article not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article by SKU",
			Handled: false,
		}
	}

	return &article, nil
}

func (r *ArticlesRepository) CreateArticle(data *requests.Article) *responses.InternalResponse {
	var existing database.Article
	err := r.DB.First(&existing, "sku = ?", data.SKU).Error
	if err == nil {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "An article with the same SKU already exists",
			Handled: true,
		}
	}

	if err != gorm.ErrRecordNotFound {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check existing article",
			Handled: false,
		}
	}

	var article database.Article
	tools.CopyStructFields(data, &article)
	article.CreatedAt = tools.GetCurrentTime()
	article.UpdatedAt = tools.GetCurrentTime()

	if article.IsActive == nil {
		trueVal := true
		article.IsActive = &trueVal
	}

	err = r.DB.Create(&article).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create article",
			Handled: false,
		}
	}

	return nil
}

func (r *ArticlesRepository) UpdateArticle(id int, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	var article database.Article
	err := r.DB.First(&article, id).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Article not found",
			Handled: true,
		}
	}

	tools.CopyStructFields(data, &article)
	article.UpdatedAt = tools.GetCurrentTime()

	err = r.DB.Save(&article).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update article",
			Handled: false,
		}
	}

	return &article, nil
}

func (r *ArticlesRepository) GetLotsBySKU(sku string) ([]database.Lot, error) {
	var lots []database.Lot
	err := r.DB.Where("sku = ?", sku).Find(&lots).Error
	return lots, err
}

func (r *ArticlesRepository) GetSerialsBySKU(sku string) ([]database.Serial, error) {
	var serials []database.Serial
	err := r.DB.Where("sku = ?", sku).Find(&serials).Error
	return serials, err
}

func (r *ArticlesRepository) ImportArticlesFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	imported := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to open Excel file",
			Handled: false,
		})
		return imported, errorsList
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to read rows",
			Handled: false,
		})
		return imported, errorsList
	}

	for i, row := range rows {
		if i < 6 {
			continue
		}

		if len(row) < 10 {
			continue
		}

		sku := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		description := strings.TrimSpace(row[2])
		priceStr := strings.TrimSpace(row[3])
		presentation := strings.TrimSpace(row[4])
		trackByLot := strings.TrimSpace(row[5]) == "Si"
		trackBySerial := strings.TrimSpace(row[6]) == "Si"
		trackExpiration := strings.TrimSpace(row[7]) == "Si"
		maxQtyStr := strings.TrimSpace(row[8])
		minQtyStr := strings.TrimSpace(row[9])

		if sku == "" || name == "" || presentation == "" {
			continue
		}

		var unitPrice *float64
		if priceStr != "" {
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				unitPrice = &p
			}
		}

		var minQty *int
		if minQtyStr != "" {
			if q, err := strconv.Atoi(minQtyStr); err == nil {
				minQty = &q
			}
		}

		var maxQty *int
		if maxQtyStr != "" {
			if q, err := strconv.Atoi(maxQtyStr); err == nil {
				maxQty = &q
			}
		}

		descPtr := &description
		if description == "" {
			descPtr = nil
		}

		article := &requests.Article{
			SKU:             sku,
			Name:            name,
			Description:     descPtr,
			UnitPrice:       unitPrice,
			Presentation:    presentation,
			TrackByLot:      trackByLot,
			TrackBySerial:   trackBySerial,
			TrackExpiration: trackExpiration,
			MinQuantity:     minQty,
			MaxQuantity:     maxQty,
			ImageURL:        nil,
		}

		resp := r.CreateArticle(article)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Row %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}

		imported = append(imported, sku)
	}

	return imported, errorsList
}

func (r *ArticlesRepository) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	articles, errResp := r.GetAllArticles()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "Nombre", "Descripción", "Precio", "Presentación",
		"Rastrear por lote", "Rastrear por serie", "Rastrear por expiración",
		"Cantidad Máxima", "Cantidad Mínima",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, article := range articles {
		row := idx + 7
		values := []interface{}{
			article.ID,
			article.Name,
			getOrEmpty(article.Description),
			getOrEmpty(article.UnitPrice),
			article.Presentation,
			boolToSiNo(article.TrackByLot),
			boolToSiNo(article.TrackBySerial),
			boolToSiNo(article.TrackExpiration),
			getOrEmpty(article.MaxQuantity),
			getOrEmpty(article.MinQuantity),
		}
		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate Excel file",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func boolToSiNo(value bool) string {
	if value {
		return "Sí"
	}
	return "No"
}

func (r *ArticlesRepository) DeleteArticle(id int) *responses.InternalResponse {
	err := r.DB.
		Table(database.Article{}.TableName()).
		Where("id = ?", id).
		Delete(&database.Article{}).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to delete article",
			Handled: false,
		}
	}

	return nil
}
