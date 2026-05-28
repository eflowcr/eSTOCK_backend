package wire

import (
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"gorm.io/gorm"
)

// NewArticleSuppliers builds ArticleSuppliersRepository (GORM-backed) and ArticleSuppliersService.
// Closes S3-5 ("Múltiples proveedores por artículo") from the 2026-04-29 scope email.
func NewArticleSuppliers(db *gorm.DB) (ports.ArticleSuppliersRepository, *services.ArticleSuppliersService) {
	r := &repositories.ArticleSuppliersRepository{DB: db}
	return r, services.NewArticleSuppliersService(r)
}
