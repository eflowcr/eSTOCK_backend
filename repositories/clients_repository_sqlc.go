package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientsRepositorySQLC implements ports.ClientsRepository using sqlc-generated queries.
type ClientsRepositorySQLC struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// NewClientsRepositorySQLC returns a clients repository backed by sqlc.
func NewClientsRepositorySQLC(queries *sqlc.Queries, pool *pgxpool.Pool) *ClientsRepositorySQLC {
	return &ClientsRepositorySQLC{queries: queries, pool: pool}
}

var _ ports.ClientsRepository = (*ClientsRepositorySQLC)(nil)

func (r *ClientsRepositorySQLC) generateID(ctx context.Context) (string, error) {
	var id string
	if err := r.pool.QueryRow(ctx, "SELECT nanoid()").Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *ClientsRepositorySQLC) Create(tenantID string, data *requests.CreateClientRequest, createdBy *string) (*database.Client, *responses.InternalResponse) {
	ctx := context.Background()

	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	id, err := r.generateID(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error generando ID", Handled: false}
	}

	arg := sqlc.CreateClientParams{
		ID:        id,
		TenantID:  tid,
		Type:      data.Type,
		Code:      data.Code,
		Name:      data.Name,
		Email:     ptrStringToPgText(data.Email),
		Phone:     ptrStringToPgText(data.Phone),
		Address:   ptrStringToPgText(data.Address),
		TaxID:     ptrStringToPgText(data.TaxID),
		Notes:     ptrStringToPgText(data.Notes),
		CreatedBy: ptrStringToPgText(createdBy),
	}
	c, err := r.queries.CreateClient(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al crear el cliente", Handled: false}
	}
	result := sqlcClientToDatabase(c)
	return &result, nil
}

// GetByID performs a lookup without tenant filter — used for internal validation only
// (e.g. picking/receiving task customer/supplier checks). Not for HTTP endpoint responses.
func (r *ClientsRepositorySQLC) GetByID(id string) (*database.Client, *responses.InternalResponse) {
	ctx := context.Background()
	c, err := r.queries.GetClientByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Cliente no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el cliente", Handled: false}
	}
	result := sqlcClientToDatabase(c)
	return &result, nil
}

// GetByIDForTenant scopes the lookup to tenantID — use for HTTP endpoint responses (HR1-M3).
func (r *ClientsRepositorySQLC) GetByIDForTenant(id, tenantID string) (*database.Client, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	c, err := r.queries.GetClientByIDForTenant(ctx, sqlc.GetClientByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Cliente no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el cliente", Handled: false}
	}
	result := sqlcClientToDatabase(c)
	return &result, nil
}

func (r *ClientsRepositorySQLC) GetByTenantAndCode(tenantID, code string) (*database.Client, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	c, err := r.queries.GetClientByTenantAndCode(ctx, sqlc.GetClientByTenantAndCodeParams{TenantID: tid, Code: code})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al buscar cliente por código", Handled: false}
	}
	result := sqlcClientToDatabase(c)
	return &result, nil
}

func (r *ClientsRepositorySQLC) ListByTenant(tenantID string) ([]database.Client, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	list, err := r.queries.ListClientsByTenant(ctx, tid)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar clientes", Handled: false}
	}
	out := make([]database.Client, len(list))
	for i, c := range list {
		out[i] = sqlcClientToDatabase(c)
	}
	return out, nil
}

func (r *ClientsRepositorySQLC) Update(id string, data *requests.UpdateClientRequest, tenantID string) (*database.Client, *responses.InternalResponse) {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	// Existence check is tenant-scoped (HR1-M3).
	_, err = r.queries.GetClientByIDForTenant(ctx, sqlc.GetClientByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Cliente no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al buscar el cliente", Handled: false}
	}
	arg := sqlc.UpdateClientParams{
		ID:       id,
		TenantID: tid,
		Type:     data.Type,
		Code:     data.Code,
		Name:     data.Name,
		Email:    ptrStringToPgText(data.Email),
		Phone:    ptrStringToPgText(data.Phone),
		Address:  ptrStringToPgText(data.Address),
		TaxID:    ptrStringToPgText(data.TaxID),
		Notes:    ptrStringToPgText(data.Notes),
	}
	c, err := r.queries.UpdateClient(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar el cliente", Handled: false}
	}
	result := sqlcClientToDatabase(c)
	return &result, nil
}

func (r *ClientsRepositorySQLC) SoftDelete(id, tenantID string) *responses.InternalResponse {
	ctx := context.Background()
	tid, err := stringToPgUUID(tenantID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "tenant_id inválido", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	// Existence check is tenant-scoped (HR1-M3).
	_, err = r.queries.GetClientByIDForTenant(ctx, sqlc.GetClientByIDForTenantParams{ID: id, TenantID: tid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &responses.InternalResponse{Message: "Cliente no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al buscar el cliente", Handled: false}
	}
	if err := r.queries.SoftDeleteClient(ctx, sqlc.SoftDeleteClientParams{ID: id, TenantID: tid}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar el cliente", Handled: false}
	}
	return nil
}

func sqlcClientToDatabase(c sqlc.Client) database.Client {
	return database.Client{
		ID:        c.ID,
		TenantID:  pgUUIDToString(c.TenantID),
		Type:      c.Type,
		Code:      c.Code,
		Name:      c.Name,
		Email:     pgTextToPtrString(c.Email),
		Phone:     pgTextToPtrString(c.Phone),
		Address:   pgTextToPtrString(c.Address),
		TaxID:     pgTextToPtrString(c.TaxID),
		Notes:     pgTextToPtrString(c.Notes),
		IsActive:  c.IsActive,
		CreatedBy: pgTextToPtrString(c.CreatedBy),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// pgUUIDToString converts pgtype.UUID to a standard UUID string.
func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// stringToPgUUID converts a UUID string to pgtype.UUID.
func stringToPgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return u, nil
}
