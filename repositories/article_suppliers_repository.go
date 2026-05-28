package repositories

import (
	"errors"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// ArticleSuppliersRepository — GORM-backed CRUD for article_suppliers (S3-5 close 2026-05-27).
// The table already has its own tenant_id column (migration 000022); no JOIN trick needed.
// UNIQUE(tenant_id, article_sku, supplier_id) prevents duplicate links.
type ArticleSuppliersRepository struct {
	DB *gorm.DB
}

// List returns the article↔supplier links for the tenant. Preferred rows surface first.
func (r *ArticleSuppliersRepository) List(tenantID string, f ports.ArticleSupplierFilters) ([]database.ArticleSupplier, *responses.InternalResponse) {
	var rows []database.ArticleSupplier
	q := r.DB.Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if f.ArticleSKU != nil && *f.ArticleSKU != "" {
		q = q.Where("article_sku = ?", *f.ArticleSKU)
	}
	if f.SupplierID != nil && *f.SupplierID != "" {
		q = q.Where("supplier_id = ?", *f.SupplierID)
	}
	if f.IsPreferred != nil {
		q = q.Where("is_preferred = ?", *f.IsPreferred)
	}
	if err := q.Order("is_preferred DESC, created_at DESC").Find(&rows).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener los proveedores del artículo", Handled: false}
	}
	return rows, nil
}

// GetByID returns a single link scoped to the tenant. 404 if missing.
func (r *ArticleSuppliersRepository) GetByID(id, tenantID string) (*database.ArticleSupplier, *responses.InternalResponse) {
	var row database.ArticleSupplier
	err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{Message: "Relación artículo-proveedor no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la relación artículo-proveedor", Handled: false}
	}
	return &row, nil
}

// Create inserts a new link. Generates the id client-side (matches the rest of the app's nanoid convention)
// and enforces "máximo un is_preferred=true por (tenant, article_sku)" in the same transaction.
// Duplicate (tenant, article, supplier) surfaces as 409 Conflict (UNIQUE constraint).
func (r *ArticleSuppliersRepository) Create(tenantID string, req *requests.CreateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse) {
	var out database.ArticleSupplier
	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		newID, idErr := tools.GenerateNanoid(tx)
		if idErr != nil {
			return idErr
		}
		isPref := req.IsPreferred != nil && *req.IsPreferred
		if isPref {
			if err := tx.Model(&database.ArticleSupplier{}).
				Where("tenant_id = ? AND article_sku = ? AND deleted_at IS NULL", tenantID, req.ArticleSKU).
				Update("is_preferred", false).Error; err != nil {
				return err
			}
		}
		out = database.ArticleSupplier{
			ID:           newID,
			TenantID:     tenantID,
			ArticleSKU:   req.ArticleSKU,
			SupplierID:   req.SupplierID,
			IsPreferred:  isPref,
			LeadTimeDays: req.LeadTimeDays,
			UnitCost:     req.UnitCost,
			SupplierSKU:  req.SupplierSKU,
			Notes:        req.Notes,
		}
		return tx.Create(&out).Error
	})
	if txErr != nil {
		if isUniqueViolation(txErr) {
			return nil, &responses.InternalResponse{Error: txErr, Message: "Ya existe esta relación artículo-proveedor", Handled: true, StatusCode: responses.StatusConflict}
		}
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al crear la relación artículo-proveedor", Handled: false}
	}
	return &out, nil
}

// Update patches mutable fields (everything except the relationship keys). Re-enforces the
// single-preferred invariant when is_preferred=true is being set.
func (r *ArticleSuppliersRepository) Update(id, tenantID string, req *requests.UpdateArticleSupplierRequest) (*database.ArticleSupplier, *responses.InternalResponse) {
	var current database.ArticleSupplier
	err := r.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{Message: "Relación artículo-proveedor no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al cargar la relación artículo-proveedor", Handled: false}
	}
	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		if req.IsPreferred != nil && *req.IsPreferred {
			if err := tx.Model(&database.ArticleSupplier{}).
				Where("tenant_id = ? AND article_sku = ? AND id <> ? AND deleted_at IS NULL", tenantID, current.ArticleSKU, id).
				Update("is_preferred", false).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{}
		if req.IsPreferred != nil {
			updates["is_preferred"] = *req.IsPreferred
		}
		if req.LeadTimeDays != nil {
			updates["lead_time_days"] = *req.LeadTimeDays
		}
		if req.UnitCost != nil {
			updates["unit_cost"] = *req.UnitCost
		}
		if req.SupplierSKU != nil {
			updates["supplier_sku"] = *req.SupplierSKU
		}
		if req.Notes != nil {
			updates["notes"] = *req.Notes
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&database.ArticleSupplier{}).
			Where("id = ? AND tenant_id = ?", id, tenantID).
			Updates(updates).Error
	})
	if txErr != nil {
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al actualizar la relación artículo-proveedor", Handled: false}
	}
	return r.GetByID(id, tenantID)
}

// SoftDelete sets deleted_at = NOW() on the row (the model has DeletedAt *time.Time, not gorm.DeletedAt,
// so soft-delete is manual). RowsAffected==0 → 404.
func (r *ArticleSuppliersRepository) SoftDelete(id, tenantID string) *responses.InternalResponse {
	res := r.DB.Model(&database.ArticleSupplier{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", gorm.Expr("NOW()"))
	if res.Error != nil {
		return &responses.InternalResponse{Error: res.Error, Message: "Error al eliminar la relación artículo-proveedor", Handled: false}
	}
	if res.RowsAffected == 0 {
		return &responses.InternalResponse{Message: "Relación artículo-proveedor no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return nil
}

// isUniqueViolation does a substring check on the driver error message; matches both pgx
// ("23505") and human-readable strings ("duplicate key", "unique constraint").
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique violation")
}
