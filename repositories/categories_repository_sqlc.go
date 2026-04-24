package repositories

import (
	"context"
	"errors"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CategoriesRepositorySQLC implements ports.CategoriesRepository using sqlc-generated queries.
type CategoriesRepositorySQLC struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// NewCategoriesRepositorySQLC returns a categories repository backed by sqlc.
func NewCategoriesRepositorySQLC(queries *sqlc.Queries, pool *pgxpool.Pool) *CategoriesRepositorySQLC {
	return &CategoriesRepositorySQLC{queries: queries, pool: pool}
}

var _ ports.CategoriesRepository = (*CategoriesRepositorySQLC)(nil)

func (r *CategoriesRepositorySQLC) generateID(ctx context.Context) (string, error) {
	var id string
	if err := r.pool.QueryRow(ctx, "SELECT nanoid()").Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *CategoriesRepositorySQLC) Create(tenantID string, data *requests.CreateCategoryRequest) (*database.Category, *responses.InternalResponse) {
	ctx := context.Background()

	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	id, err := r.generateID(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error generando ID", Handled: false}
	}

	arg := sqlc.CreateCategoryParams{
		ID:       id,
		TenantID: tid,
		Name:     data.Name,
		ParentID: ptrStringToPgText(data.ParentID),
	}
	c, err := r.queries.CreateCategory(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al crear la categoría", Handled: false}
	}
	result := sqlcCategoryToDatabase(c)
	return &result, nil
}

func (r *CategoriesRepositorySQLC) GetByID(id string) (*database.Category, *responses.InternalResponse) {
	ctx := context.Background()
	c, err := r.queries.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Categoría no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la categoría", Handled: false}
	}
	result := sqlcCategoryToDatabase(c)
	return &result, nil
}

// ListByTenant returns all categories for the tenant without optional filters.
// Internally delegates to ListByTenantFiltered with nil params (M8).
func (r *CategoriesRepositorySQLC) ListByTenant(tenantID string) ([]database.Category, *responses.InternalResponse) {
	return r.ListByTenantFiltered(tenantID, nil, nil, nil, nil)
}

// ListByTenantFiltered pushes optional isActive/search filters and pagination to SQL (M8 — HR1 deferred).
// Pass nil for any param to skip that filter. limit/offset default to 200/0 in SQL.
func (r *CategoriesRepositorySQLC) ListByTenantFiltered(tenantID string, isActive *bool, search *string, limit *int32, offset *int32) ([]database.Category, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	arg := sqlc.ListCategoriesByTenantParams{
		TenantID: tid,
		IsActive: ptrBoolToPgBool(isActive),
		Search:   ptrStringToPgText(search),
		Limit:    ptrInt32ToPgInt4(limit),
		Offset:   ptrInt32ToPgInt4(offset),
	}
	list, err := r.queries.ListCategoriesByTenant(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar categorías", Handled: false}
	}
	out := make([]database.Category, len(list))
	for i, c := range list {
		out[i] = sqlcCategoryToDatabase(c)
	}
	return out, nil
}

func (r *CategoriesRepositorySQLC) Update(id string, data *requests.UpdateCategoryRequest) (*database.Category, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Categoría no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al buscar la categoría", Handled: false}
	}
	arg := sqlc.UpdateCategoryParams{
		ID:       id,
		Name:     data.Name,
		ParentID: ptrStringToPgText(data.ParentID),
	}
	c, err := r.queries.UpdateCategory(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar la categoría", Handled: false}
	}
	result := sqlcCategoryToDatabase(c)
	return &result, nil
}

func (r *CategoriesRepositorySQLC) SoftDelete(id string) *responses.InternalResponse {
	ctx := context.Background()
	_, err := r.queries.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{Message: "Categoría no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al buscar la categoría", Handled: false}
	}
	if err := r.queries.SoftDeleteCategory(ctx, id); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la categoría", Handled: false}
	}
	return nil
}

func sqlcCategoryToDatabase(c sqlc.Category) database.Category {
	return database.Category{
		ID:        c.ID,
		TenantID:  pgUUIDToString(c.TenantID),
		Name:      c.Name,
		ParentID:  pgTextToPtrString(c.ParentID),
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
