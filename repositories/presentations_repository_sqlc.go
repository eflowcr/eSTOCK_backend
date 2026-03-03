package repositories

import (
	"context"
	"errors"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PresentationsRepositorySQLC implements ports.PresentationsRepository using sqlc-generated queries.
type PresentationsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewPresentationsRepositorySQLC returns a presentations repository backed by sqlc.
func NewPresentationsRepositorySQLC(queries *sqlc.Queries) *PresentationsRepositorySQLC {
	return &PresentationsRepositorySQLC{queries: queries}
}

var _ ports.PresentationsRepository = (*PresentationsRepositorySQLC)(nil)

func (r *PresentationsRepositorySQLC) GetAllPresentations() ([]database.Presentations, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListPresentations(ctx)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las presentaciones", Handled: false}
	}
	out := make([]database.Presentations, len(list))
	for i, p := range list {
		out[i] = sqlcPresentationToDatabase(p)
	}
	return out, nil
}

func (r *PresentationsRepositorySQLC) GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse) {
	ctx := context.Background()
	p, err := r.queries.GetPresentationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Presentación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la presentación", Handled: false}
	}
	pres := sqlcPresentationToDatabase(p)
	return &pres, nil
}

func (r *PresentationsRepositorySQLC) CreatePresentation(data *database.Presentations) *responses.InternalResponse {
	ctx := context.Background()
	exists, err := r.queries.PresentationExistsByID(ctx, data.PresentationId)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al verificar la existencia de la presentación", Handled: false}
	}
	if exists {
		return &responses.InternalResponse{
			Message:    "Ya existe una presentación con el ID proporcionado",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	arg := sqlc.CreatePresentationParams{
		PresentationID: data.PresentationId,
		Description:    stringToPgText(data.Description),
	}
	_, err = r.queries.CreatePresentation(ctx, arg)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear la presentación", Handled: false}
	}
	return nil
}

func (r *PresentationsRepositorySQLC) UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetPresentationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Presentación no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener la presentación", Handled: false}
	}

	arg := sqlc.UpdatePresentationParams{
		PresentationID: id,
		Description:    stringToPgText(name),
	}
	updated, err := r.queries.UpdatePresentation(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar la presentación", Handled: false}
	}
	pres := sqlcPresentationToDatabase(updated)
	return &pres, nil
}

func (r *PresentationsRepositorySQLC) DeletePresentation(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeletePresentation(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar la presentación", Handled: false}
	}
	return nil
}

func sqlcPresentationToDatabase(p sqlc.Presentation) database.Presentations {
	return database.Presentations{
		PresentationId: p.PresentationID,
		Description:    pgTextToString(p.Description),
	}
}

func stringToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
