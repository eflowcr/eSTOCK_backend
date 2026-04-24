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

// categoryLookupForArticles is a narrow read-only interface for category validation.
type categoryLookupForArticles interface {
	GetByID(id string) (*database.Category, *responses.InternalResponse)
}

// locationLookupForArticles is a narrow read-only interface for location validation.
type locationLookupForArticles interface {
	GetLocationByID(id string) (*database.Location, *responses.InternalResponse)
}

// ArticlesService — S3.5 W1: every public method now requires a tenantID. Controllers
// pass it from Config.TenantID (env-injected); a future M2 wave will source it from
// the JWT claim instead so multi-tenant signup can work end-to-end.
type ArticlesService struct {
	Repository     ports.ArticlesRepository
	CategoriesRepo categoryLookupForArticles // optional: validate category_id on create/update
	LocationsRepo  locationLookupForArticles // optional: validate default_location_id
}

func NewArticlesService(repo ports.ArticlesRepository) *ArticlesService {
	return &ArticlesService{
		Repository: repo,
	}
}

// WithCategoriesRepo attaches a CategoriesRepository for category_id validation.
func (s *ArticlesService) WithCategoriesRepo(r categoryLookupForArticles) *ArticlesService {
	s.CategoriesRepo = r
	return s
}

// WithLocationsRepo attaches a LocationsRepository for default_location_id validation.
func (s *ArticlesService) WithLocationsRepo(r locationLookupForArticles) *ArticlesService {
	s.LocationsRepo = r
	return s
}

func (s *ArticlesService) GetAllArticles(tenantID string) ([]database.Article, *responses.InternalResponse) {
	return s.Repository.GetAllArticlesForTenant(tenantID)
}

func (s *ArticlesService) GetArticleByID(id, tenantID string) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetArticleByIDForTenant(id, tenantID)
}

func (s *ArticlesService) GetBySku(sku, tenantID string) (*database.Article, *responses.InternalResponse) {
	return s.Repository.GetBySkuForTenant(sku, tenantID)
}

// EnrichArticle builds an ArticleResponse with embedded category and default_location objects.
func (s *ArticlesService) EnrichArticle(art *database.Article) *responses.ArticleResponse {
	if art == nil {
		return nil
	}
	r := &responses.ArticleResponse{
		ID:                 art.ID,
		SKU:                art.SKU,
		Name:               art.Name,
		Description:        art.Description,
		UnitPrice:          art.UnitPrice,
		Presentation:       art.Presentation,
		TrackByLot:         art.TrackByLot,
		TrackBySerial:      art.TrackBySerial,
		TrackExpiration:    art.TrackExpiration,
		RotationStrategy:   art.RotationStrategy,
		MinQuantity:        art.MinQuantity,
		MaxQuantity:        art.MaxQuantity,
		ImageURL:           art.ImageURL,
		IsActive:           art.IsActive,
		CreatedAt:          art.CreatedAt,
		UpdatedAt:          art.UpdatedAt,
		CategoryID:         art.CategoryID,
		ShelfLifeInDays:    art.ShelfLifeInDays,
		SafetyStock:        art.SafetyStock,
		BatchNumberSeries:  art.BatchNumberSeries,
		SerialNumberSeries: art.SerialNumberSeries,
		MinOrderQty:        art.MinOrderQty,
		DefaultLocationID:  art.DefaultLocationID,
		ReceivingNotes:     art.ReceivingNotes,
		ShippingNotes:      art.ShippingNotes,
	}
	if art.CategoryID != nil && s.CategoriesRepo != nil {
		cat, _ := s.CategoriesRepo.GetByID(*art.CategoryID)
		if cat != nil {
			r.Category = &responses.EmbeddedCategory{ID: cat.ID, Name: cat.Name}
		}
	}
	if art.DefaultLocationID != nil && s.LocationsRepo != nil {
		loc, _ := s.LocationsRepo.GetLocationByID(*art.DefaultLocationID)
		if loc != nil {
			r.DefaultLocation = &responses.EmbeddedLocation{ID: loc.ID, Code: loc.LocationCode}
		}
	}
	return r
}

func (s *ArticlesService) CreateArticle(tenantID string, article *requests.Article) *responses.InternalResponse {
	if errResp := s.validateArticleFields(article); errResp != nil {
		return errResp
	}
	if errResp := s.validateRotationStrategy(article.RotationStrategy, article.TrackExpiration); errResp != nil {
		return errResp
	}
	resp := s.Repository.CreateArticleForTenant(tenantID, article)
	if resp != nil && resp.Error != nil && !resp.Handled {
		tools.LogServiceError("articles", "CreateArticle", resp.Error, resp.Message)
	}
	return resp
}

func (s *ArticlesService) UpdateArticle(id, tenantID string, data *requests.Article) (*database.Article, *responses.InternalResponse, []map[string]interface{}) {
	article, errResp := s.Repository.GetArticleByIDForTenant(id, tenantID)
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

	if errResp := s.validateArticleFields(data); errResp != nil {
		return nil, errResp, nil
	}

	if errResp := s.validateRotationStrategy(data.RotationStrategy, data.TrackExpiration); errResp != nil {
		return nil, errResp, nil
	}

	updated, errResp := s.Repository.UpdateArticleForTenant(id, tenantID, data)
	return updated, errResp, warnings
}

func (s *ArticlesService) ImportArticlesFromExcel(tenantID string, fileBytes []byte) ([]string, []string, []*responses.InternalResponse) {
	return s.Repository.ImportArticlesFromExcelForTenant(tenantID, fileBytes)
}

func (s *ArticlesService) ImportArticlesFromJSON(tenantID string, rows []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	return s.Repository.ImportArticlesFromJSONForTenant(tenantID, rows)
}

func (s *ArticlesService) ExportArticlesToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportArticlesToExcelForTenant(tenantID)
}

func (s *ArticlesService) ValidateImportRows(tenantID string, rows []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	return s.Repository.ValidateImportRowsForTenant(tenantID, rows)
}

func (s *ArticlesService) GenerateImportTemplate(tenantID, language string) ([]byte, *responses.InternalResponse) {
	return s.Repository.GenerateImportTemplateForTenant(tenantID, language)
}

func (s *ArticlesService) DeleteArticle(id, tenantID string) *responses.InternalResponse {
	return s.Repository.DeleteArticleForTenant(id, tenantID)
}

// validateArticleFields validates the new M2 fields.
func (s *ArticlesService) validateArticleFields(data *requests.Article) *responses.InternalResponse {
	if data.ShelfLifeInDays != nil && *data.ShelfLifeInDays < 0 {
		return &responses.InternalResponse{
			Message:    "shelf_life_in_days debe ser >= 0",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if data.SafetyStock < 0 {
		return &responses.InternalResponse{
			Message:    "safety_stock debe ser >= 0",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if data.MinOrderQty < 0 {
		return &responses.InternalResponse{
			Message:    "min_order_qty debe ser >= 0",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if data.CategoryID != nil && *data.CategoryID != "" && s.CategoriesRepo != nil {
		cat, resp := s.CategoriesRepo.GetByID(*data.CategoryID)
		if resp != nil || cat == nil {
			return &responses.InternalResponse{
				Message:    fmt.Sprintf("category_id inválido: categoría no encontrada"),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
	}
	if data.DefaultLocationID != nil && *data.DefaultLocationID != "" && s.LocationsRepo != nil {
		loc, resp := s.LocationsRepo.GetLocationByID(*data.DefaultLocationID)
		if resp != nil || loc == nil {
			return &responses.InternalResponse{
				Message:    "default_location_id inválido: ubicación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
	}
	return nil
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
