package responses

import "github.com/eflowcr/eSTOCK_backend/models/requests"

// ArticleValidationStatus describes the result of validating one import row against the DB.
type ArticleValidationStatus string

const (
	ArticleStatusNew       ArticleValidationStatus = "new"
	ArticleStatusExists    ArticleValidationStatus = "exists"
	ArticleStatusSimilar   ArticleValidationStatus = "similar"
	ArticleStatusError     ArticleValidationStatus = "error"
	ArticleStatusDuplicate ArticleValidationStatus = "duplicate"
)

// ArticleValidationMatch is a compact representation of an existing DB article used in conflict reports.
type ArticleValidationMatch struct {
	ID           string `json:"id"`
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Presentation string `json:"presentation"`
	IsActive     bool   `json:"is_active"`
}

// ArticleValidationResult is the per-row output of the validate endpoint.
type ArticleValidationResult struct {
	RowIndex        int                       `json:"row_index"`
	Status          ArticleValidationStatus   `json:"status"`
	Row             requests.ArticleImportRow `json:"row"`
	FieldErrors     map[string]string         `json:"field_errors,omitempty"`
	ExistingArticle *ArticleValidationMatch   `json:"existing_article,omitempty"`
	SimilarArticles []ArticleValidationMatch  `json:"similar_articles,omitempty"`
}
