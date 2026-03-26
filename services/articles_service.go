package services

import (
	"fmt"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
)

type ArticlesService struct {
	Repository ports.ArticlesRepository
}

func NewArticlesService(repo ports.ArticlesRepository) *ArticlesService {
	return &ArticlesService{
		Repository: repo,
	}
}

func (s *ArticlesService) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	return s.Repository.GetAllArticles()
}

func (s *ArticlesService) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetArticleByID(id)
}

func (s *ArticlesService) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetBySku(sku)
}

func (s *ArticlesService) CreateArticle(article *requests.Article) *responses.InternalResponse {
	if errResp := s.validateRotationStrategy(article.RotationStrategy, article.TrackExpiration); errResp != nil {
		return errResp
	}
	resp := s.Repository.CreateArticle(article)
	if resp != nil && resp.Error != nil && !resp.Handled {
		tools.LogServiceError("articles", "CreateArticle", resp.Error, resp.Message)
	}
	return resp
}

func (s *ArticlesService) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse, []map[string]interface{}) {
	article, errResp := s.Repository.GetArticleByID(id)
	if errResp != nil {
		return nil, errResp, nil
	}

	warnings := []map[string]interface{}{}

	lotTrackingDisabled := article.TrackByLot && !data.TrackByLot
	serialTrackingDisabled := article.TrackBySerial && !data.TrackBySerial

	if lotTrackingDisabled {
		lots, err := s.Repository.GetLotsBySKU(article.SKU)
		if err == nil && len(lots) > 0 {
			warnings = append(warnings, map[string]interface{}{
				"type":    "lot_tracking_disabled",
				"count":   len(lots),
				"message": fmt.Sprintf("Warning: %d existing lot record(s) found. Disabling lot tracking will make this data inaccessible through the system, but it will remain in the database.", len(lots)),
			})
		}
	}

	if serialTrackingDisabled {
		serials, err := s.Repository.GetSerialsBySKU(article.SKU)
		if err == nil && len(serials) > 0 {
			warnings = append(warnings, map[string]interface{}{
				"type":    "serial_tracking_disabled",
				"count":   len(serials),
				"message": fmt.Sprintf("Warning: %d existing serial record(s) found. Disabling serial tracking will make this data inaccessible through the system, but it will remain in the database.", len(serials)),
			})
		}
	}

	if errResp := s.validateRotationStrategy(data.RotationStrategy, data.TrackExpiration); errResp != nil {
		return nil, errResp, nil
	}

	updated, errResp := s.Repository.UpdateArticle(id, data)
	return updated, errResp, warnings
}

func (s *ArticlesService) ImportArticlesFromExcel(fileBytes []byte) ([]string, []string, []*responses.InternalResponse) {
	return s.Repository.ImportArticlesFromExcel(fileBytes)
}

func (s *ArticlesService) ImportArticlesFromJSON(rows []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	return s.Repository.ImportArticlesFromJSON(rows)
}

func (s *ArticlesService) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportArticlesToExcel()
}

func (s *ArticlesService) GenerateImportTemplate(language string) ([]byte, *responses.InternalResponse) {
	return s.Repository.GenerateImportTemplate(language)
}

func (s *ArticlesService) DeleteArticle(id string) *responses.InternalResponse {
	return s.Repository.DeleteArticle(id)
}

// validateRotationStrategy enforces WMS rule: FEFO requires expiration tracking.
func (s *ArticlesService) validateRotationStrategy(rotationStrategy string, trackExpiration bool) *responses.InternalResponse {
	rs := strings.TrimSpace(strings.ToLower(rotationStrategy))
	if rs != "fefo" {
		return nil
	}
	if !trackExpiration {
		return &responses.InternalResponse{
			Message:    "FEFO (First Expiry, First Out) requires expiration tracking to be enabled. Enable 'Track expiration' or use FIFO.",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return nil
}
