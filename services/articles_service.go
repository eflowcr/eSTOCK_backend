package services

import (
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type ArticlesService struct {
	Repository *repositories.ArticlesRepository
}

func NewArticlesService(repo *repositories.ArticlesRepository) *ArticlesService {
	return &ArticlesService{
		Repository: repo,
	}
}

func (s *ArticlesService) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	return s.Repository.GetAllArticles()
}

func (s *ArticlesService) GetArticleByID(id int) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetArticleByID(id)
}

func (s *ArticlesService) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetBySku(sku)
}

func (s *ArticlesService) CreateArticle(article *requests.Article) *responses.InternalResponse {
	return s.Repository.CreateArticle(article)
}

func (s *ArticlesService) UpdateArticle(id int, data *requests.Article) (*database.Article, *responses.InternalResponse, []map[string]interface{}) {
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

	updated, errResp := s.Repository.UpdateArticle(id, data)
	return updated, errResp, warnings
}

func (s *ArticlesService) ImportArticlesFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportArticlesFromExcel(fileBytes)
}

func (s *ArticlesService) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportArticlesToExcel()
}

func (s *ArticlesService) DeleteArticle(id int) *responses.InternalResponse {
	return s.Repository.DeleteArticle(id)
}
