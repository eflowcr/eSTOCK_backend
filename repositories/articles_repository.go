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
			Message: "Error al obtener los artículos",
			Handled: false,
		}
	}

	return articles, nil
}

func (r *ArticlesRepository) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	var article database.Article

	err := r.DB.
		Table(database.Article{}.TableName()).
		Where("id = ?", id).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el artículo",
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
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el artículo por SKU",
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
			Message:    "Ya existe un artículo con el mismo SKU",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	if err != gorm.ErrRecordNotFound {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al verificar el artículo existente",
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
			Message: "Error al crear el artículo",
			Handled: false,
		}
	}

	return nil
}

func (r *ArticlesRepository) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	var article database.Article
	err := r.DB.First(&article, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el artículo",
			Handled: false,
		}
	}

	tools.CopyStructFields(data, &article)
	article.UpdatedAt = tools.GetCurrentTime()

	err = r.DB.Save(&article).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar el artículo",
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

func (r *ArticlesRepository) ImportArticlesFromExcel(fileBytes []byte) ([]string, []string, []*responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al abrir el archivo de Excel",
			Handled: false,
		})
		return imported, skipped, errorsList
	}

	// Use first sheet regardless of language-based name ("Artículos", "Articles", "Sheet1")
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		errorsList = append(errorsList, &responses.InternalResponse{
			Message: "El archivo no contiene hojas de datos",
			Handled: true,
		})
		return imported, skipped, errorsList
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet)
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al leer las filas de Excel",
			Handled: false,
		})
		return imported, skipped, errorsList
	}

	for i, row := range rows {
		// Skip header, instructions, column-header row, and example row (rows 1-8, index 0-7)
		if i < 8 {
			continue
		}
		if len(row) < 10 {
			continue
		}

		sku := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])

		// Skip rows where required fields are empty
		if sku == "" || name == "" {
			continue
		}

		// Detect and skip example row gracefully
		if strings.EqualFold(sku, "ART-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
			continue
		}

		description := strings.TrimSpace(row[2])
		priceStr := strings.TrimSpace(row[3])
		presentation := strings.TrimSpace(row[4])
		trackByLot := parseBoolCell(strings.TrimSpace(row[5]))
		trackBySerial := parseBoolCell(strings.TrimSpace(row[6]))
		trackExpiration := parseBoolCell(strings.TrimSpace(row[7]))
		maxQtyStr := strings.TrimSpace(row[8])
		minQtyStr := strings.TrimSpace(row[9])

		if presentation == "" {
			errorsList = append(errorsList, &responses.InternalResponse{
				Message: fmt.Sprintf("Fila %d: presentación requerida", i+1),
				Handled: true,
			})
			continue
		}

		rotationStrategy := ""
		if len(row) > 10 {
			rs := strings.ToLower(strings.TrimSpace(row[10]))
			if rs == "fifo" || rs == "fefo" {
				rotationStrategy = rs
			}
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
			SKU:              sku,
			Name:             name,
			Description:      descPtr,
			UnitPrice:        unitPrice,
			Presentation:     presentation,
			TrackByLot:       trackByLot,
			TrackBySerial:    trackBySerial,
			TrackExpiration:  trackExpiration,
			RotationStrategy: rotationStrategy,
			MinQuantity:      minQty,
			MaxQuantity:      maxQty,
			ImageURL:         nil,
		}

		resp := r.CreateArticle(article)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}

		imported = append(imported, sku)
	}

	return imported, skipped, errorsList
}

// ImportArticlesFromJSON imports articles from a pre-validated JSON payload (used by the frontend preview flow).
func (r *ArticlesRepository) ValidateImportRows(rows []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	results := make([]responses.ArticleValidationResult, 0, len(rows))
	seenSKUs := make(map[string]bool)

	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		name := strings.TrimSpace(row.Name)
		result := responses.ArticleValidationResult{
			RowIndex: i,
			Row:      row,
		}

		// Field validation
		if sku == "" || name == "" || strings.TrimSpace(row.Presentation) == "" {
			result.Status = responses.ArticleStatusError
			result.FieldErrors = map[string]string{}
			if sku == "" {
				result.FieldErrors["sku"] = "SKU requerido"
			}
			if name == "" {
				result.FieldErrors["name"] = "Nombre requerido"
			}
			if strings.TrimSpace(row.Presentation) == "" {
				result.FieldErrors["presentation"] = "Presentación requerida"
			}
			results = append(results, result)
			continue
		}

		// Duplicate within batch
		skuKey := strings.ToLower(sku)
		if seenSKUs[skuKey] {
			result.Status = responses.ArticleStatusDuplicate
			results = append(results, result)
			continue
		}
		seenSKUs[skuKey] = true

		// Exact SKU match in DB
		existing, _ := r.GetBySku(sku)
		if existing != nil {
			isActive := false
			if existing.IsActive != nil {
				isActive = *existing.IsActive
			}
			result.Status = responses.ArticleStatusExists
			result.ExistingArticle = &responses.ArticleValidationMatch{
				ID:           existing.ID,
				SKU:          existing.SKU,
				Name:         existing.Name,
				Presentation: existing.Presentation,
				IsActive:     isActive,
			}
			results = append(results, result)
			continue
		}

		// Similar name check (LIKE search)
		keyword := name
		if len(keyword) > 20 {
			keyword = keyword[:20]
		}
		var similar []database.Article
		r.DB.Where("LOWER(name) LIKE LOWER(?) AND sku != ?", "%"+keyword+"%", sku).Limit(3).Find(&similar)
		if len(similar) > 0 {
			result.Status = responses.ArticleStatusSimilar
			result.SimilarArticles = make([]responses.ArticleValidationMatch, 0, len(similar))
			for _, s := range similar {
				isActive := false
				if s.IsActive != nil {
					isActive = *s.IsActive
				}
				result.SimilarArticles = append(result.SimilarArticles, responses.ArticleValidationMatch{
					ID:           s.ID,
					SKU:          s.SKU,
					Name:         s.Name,
					Presentation: s.Presentation,
					IsActive:     isActive,
				})
			}
			results = append(results, result)
			continue
		}

		result.Status = responses.ArticleStatusNew
		results = append(results, result)
	}

	return results, nil
}

func (r *ArticlesRepository) ImportArticlesFromJSON(rows []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}
	errorsList := []*responses.InternalResponse{}

	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		name := strings.TrimSpace(row.Name)

		if sku == "" || name == "" {
			skipped = append(skipped, fmt.Sprintf("Fila %d: SKU y nombre son requeridos", i+1))
			continue
		}
		if strings.EqualFold(sku, "ART-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
			continue
		}

		presentation := strings.TrimSpace(row.Presentation)
		if presentation == "" {
			errorsList = append(errorsList, &responses.InternalResponse{
				Message: fmt.Sprintf("Fila %d: presentación requerida", i+1),
				Handled: true,
			})
			continue
		}

		rotationStrategy := ""
		rs := strings.ToLower(strings.TrimSpace(row.RotationStrategy))
		if rs == "fifo" || rs == "fefo" {
			rotationStrategy = rs
		}

		var unitPrice *float64
		if p, err := strconv.ParseFloat(strings.TrimSpace(row.UnitPrice), 64); err == nil {
			unitPrice = &p
		}
		var minQty *int
		if q, err := strconv.Atoi(strings.TrimSpace(row.MinQuantity)); err == nil {
			minQty = &q
		}
		var maxQty *int
		if q, err := strconv.Atoi(strings.TrimSpace(row.MaxQuantity)); err == nil {
			maxQty = &q
		}

		desc := strings.TrimSpace(row.Description)
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}

		article := &requests.Article{
			SKU:              sku,
			Name:             name,
			Description:      descPtr,
			UnitPrice:        unitPrice,
			Presentation:     presentation,
			TrackByLot:       parseBoolCell(row.TrackByLot),
			TrackBySerial:    parseBoolCell(row.TrackBySerial),
			TrackExpiration:  parseBoolCell(row.TrackExpiration),
			RotationStrategy: rotationStrategy,
			MinQuantity:      minQty,
			MaxQuantity:      maxQty,
		}

		resp := r.CreateArticle(article)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}
		imported = append(imported, sku)
	}

	return imported, skipped, errorsList
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
		"SKU", "Nombre", "Descripción", "Precio", "Presentación",
		"Rastrear por lote", "Rastrear por serie", "Rastrear por expiración",
		"Cantidad Máxima", "Cantidad Mínima", "Activo",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, article := range articles {
		row := idx + 7
		isActive := false
		if article.IsActive != nil {
			isActive = *article.IsActive
		}
		values := []interface{}{
			article.SKU,
			article.Name,
			getOrEmpty(article.Description),
			getOrEmpty(article.UnitPrice),
			article.Presentation,
			boolToSiNo(article.TrackByLot),
			boolToSiNo(article.TrackBySerial),
			boolToSiNo(article.TrackExpiration),
			getOrEmpty(article.MaxQuantity),
			getOrEmpty(article.MinQuantity),
			boolToSiNo(isActive),
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
			Message: "Error al generar el archivo de Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func (r *ArticlesRepository) GenerateImportTemplate(language string) ([]byte, *responses.InternalResponse) {
	var presentations []string
	r.DB.Table("articles").Distinct("presentation").Pluck("presentation", &presentations)
	return buildImportTemplate(presentations, language)
}

func buildImportTemplate(presentations []string, language string) ([]byte, *responses.InternalResponse) {
	l := getLang(language)
	dataSheet := l["sheet_data"]

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", dataSheet)

	if err := applyArticleTemplateHeader(f, dataSheet, language); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar el encabezado de la plantilla",
			Handled: false,
		}
	}

	if err := applyArticleTemplateColumnHeaders(f, dataSheet, language); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar los encabezados de columna",
			Handled: false,
		}
	}

	if err := applyArticleTemplateValidations(f, dataSheet, presentations, language); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al aplicar validaciones a la plantilla",
			Handled: false,
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al escribir la plantilla de importación",
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

func parseBoolCell(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "si" || s == "sí" || s == "yes" || s == "true" || s == "1"
}

func (r *ArticlesRepository) DeleteArticle(id string) *responses.InternalResponse {
	err := r.DB.
		Table(database.Article{}.TableName()).
		Where("id = ?", id).
		Delete(&database.Article{}).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar el artículo",
			Handled: false,
		}
	}

	return nil
}
