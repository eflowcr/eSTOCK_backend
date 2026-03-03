package repositories

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"
)

// ArticlesRepositorySQLC implements ports.ArticlesRepository using sqlc-generated queries.
type ArticlesRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewArticlesRepositorySQLC returns an articles repository backed by sqlc.
func NewArticlesRepositorySQLC(queries *sqlc.Queries) *ArticlesRepositorySQLC {
	return &ArticlesRepositorySQLC{queries: queries}
}

// Ensure ArticlesRepositorySQLC implements ports.ArticlesRepository at compile time.
var _ ports.ArticlesRepository = (*ArticlesRepositorySQLC)(nil)

func (r *ArticlesRepositorySQLC) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListArticles(ctx)
	if err != nil {
		tools.LogRepoError("articles", "ListArticles", err, "Error al obtener los artículos")
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener los artículos", Handled: false}
	}
	out := make([]database.Article, len(list))
	for i, a := range list {
		out[i] = sqlcArticleToDatabase(a)
	}
	return out, nil
}

func (r *ArticlesRepositorySQLC) GetArticleByID(id int) (*database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	a, err := r.queries.GetArticleByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		tools.LogRepoError("articles", "GetArticleByID", err, "Error al obtener el artículo")
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el artículo", Handled: false}
	}
	art := sqlcArticleToDatabase(a)
	return &art, nil
}

func (r *ArticlesRepositorySQLC) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	a, err := r.queries.GetArticleBySku(ctx, sku)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		tools.LogRepoError("articles", "GetBySku", err, "Error al obtener el artículo por SKU")
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el artículo por SKU", Handled: false}
	}
	art := sqlcArticleToDatabase(a)
	return &art, nil
}

func (r *ArticlesRepositorySQLC) CreateArticle(data *requests.Article) *responses.InternalResponse {
	ctx := context.Background()
	exists, err := r.queries.ArticleExistsBySku(ctx, data.SKU)
	if err != nil {
		tools.LogRepoError("articles", "CreateArticle", err, "Error al verificar el artículo existente")
		return &responses.InternalResponse{Error: err, Message: "Error al verificar el artículo existente", Handled: false}
	}
	if exists {
		return &responses.InternalResponse{
			Message:    "Ya existe un artículo con el mismo SKU",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	arg := sqlc.CreateArticleParams{
		Sku:             data.SKU,
		Name:            data.Name,
		Description:     ptrStringToPgText(data.Description),
		UnitPrice:       ptrFloatToPgNumeric(data.UnitPrice),
		Presentation:    data.Presentation,
		TrackByLot:      data.TrackByLot,
		TrackBySerial:   data.TrackBySerial,
		TrackExpiration: data.TrackExpiration,
		MinQuantity:     ptrIntToPgInt4(data.MinQuantity),
		MaxQuantity:     ptrIntToPgInt4(data.MaxQuantity),
		ImageUrl:        ptrStringToPgText(data.ImageURL),
	}

	_, err = r.queries.CreateArticle(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &responses.InternalResponse{
				Message:    "Ya existe un artículo con el mismo SKU",
				Handled:    true,
				StatusCode: responses.StatusConflict,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al crear el artículo", Handled: false}
	}
	return nil
}

func (r *ArticlesRepositorySQLC) UpdateArticle(id int, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	existing, err := r.queries.GetArticleByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{
				Message:    "Artículo no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el artículo", Handled: false}
	}

	arg := sqlc.UpdateArticleParams{
		ID:              existing.ID,
		Sku:             data.SKU,
		Name:            data.Name,
		Description:     ptrStringToPgText(data.Description),
		UnitPrice:       ptrFloatToPgNumeric(data.UnitPrice),
		Presentation:    data.Presentation,
		TrackByLot:      data.TrackByLot,
		TrackBySerial:   data.TrackBySerial,
		TrackExpiration: data.TrackExpiration,
		MinQuantity:     ptrIntToPgInt4(data.MinQuantity),
		MaxQuantity:     ptrIntToPgInt4(data.MaxQuantity),
		ImageUrl:        ptrStringToPgText(data.ImageURL),
		IsActive:        existing.IsActive,
	}

	updated, err := r.queries.UpdateArticle(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar el artículo", Handled: false}
	}
	art := sqlcArticleToDatabase(updated)
	return &art, nil
}

func (r *ArticlesRepositorySQLC) GetLotsBySKU(sku string) ([]database.Lot, error) {
	ctx := context.Background()
	list, err := r.queries.ListLotsBySku(ctx, sku)
	if err != nil {
		return nil, err
	}
	out := make([]database.Lot, len(list))
	for i, l := range list {
		out[i] = sqlcLotToDatabase(l)
	}
	return out, nil
}

func (r *ArticlesRepositorySQLC) GetSerialsBySKU(sku string) ([]database.Serial, error) {
	ctx := context.Background()
	list, err := r.queries.ListSerialsBySku(ctx, sku)
	if err != nil {
		return nil, err
	}
	out := make([]database.Serial, len(list))
	for i, s := range list {
		out[i] = sqlcSerialToDatabase(s)
	}
	return out, nil
}

func (r *ArticlesRepositorySQLC) DeleteArticle(id int) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeleteArticle(ctx, int32(id))
	if err != nil {
		tools.LogRepoError("articles", "DeleteArticle", err, "Error al eliminar el artículo")
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar el artículo", Handled: false}
	}
	return nil
}

func (r *ArticlesRepositorySQLC) ImportArticlesFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	imported := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al abrir el archivo de Excel",
			Handled: false,
		})
		return imported, errorsList
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al leer las filas de Excel",
			Handled: false,
		})
		return imported, errorsList
	}

	for i, row := range rows {
		if i < 6 {
			continue
		}
		if len(row) < 10 {
			continue
		}

		sku := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		description := strings.TrimSpace(row[2])
		priceStr := strings.TrimSpace(row[3])
		presentation := strings.TrimSpace(row[4])
		trackByLot := strings.TrimSpace(row[5]) == "Si"
		trackBySerial := strings.TrimSpace(row[6]) == "Si"
		trackExpiration := strings.TrimSpace(row[7]) == "Si"
		maxQtyStr := strings.TrimSpace(row[8])
		minQtyStr := strings.TrimSpace(row[9])

		if sku == "" || name == "" || presentation == "" {
			continue
		}

		var unitPrice *float64
		if priceStr != "" {
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				unitPrice = &p
			}
		}
		var minQty, maxQty *int
		if minQtyStr != "" {
			if q, err := strconv.Atoi(minQtyStr); err == nil {
				minQty = &q
			}
		}
		if maxQtyStr != "" {
			if q, err := strconv.Atoi(maxQtyStr); err == nil {
				maxQty = &q
			}
		}
		var descPtr *string
		if description != "" {
			descPtr = &description
		}

		article := &requests.Article{
			SKU:             sku,
			Name:            name,
			Description:     descPtr,
			UnitPrice:       unitPrice,
			Presentation:    presentation,
			TrackByLot:      trackByLot,
			TrackBySerial:   trackBySerial,
			TrackExpiration: trackExpiration,
			MinQuantity:     minQty,
			MaxQuantity:     maxQty,
			ImageURL:        nil,
		}

		resp := r.CreateArticle(article)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Row %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}
		imported = append(imported, sku)
	}
	return imported, errorsList
}

func (r *ArticlesRepositorySQLC) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	articles, errResp := r.GetAllArticles()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "Nombre", "Descripción", "Precio", "Presentación",
		"Rastrear por lote", "Rastrear por serie", "Rastrear por expiración",
		"Cantidad Máxima", "Cantidad Mínima",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, article := range articles {
		row := idx + 7
		values := []interface{}{
			article.ID,
			article.Name,
			getOrEmpty(article.Description),
			getOrEmpty(article.UnitPrice),
			article.Presentation,
			boolToSiNo(article.TrackByLot),
			boolToSiNo(article.TrackBySerial),
			boolToSiNo(article.TrackExpiration),
			getOrEmpty(article.MaxQuantity),
			getOrEmpty(article.MinQuantity),
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

// --- mapping helpers ---

func sqlcArticleToDatabase(a sqlc.Article) database.Article {
	return database.Article{
		ID:              int(a.ID),
		SKU:             a.Sku,
		Name:            a.Name,
		Description:     pgTextToPtrString(a.Description),
		UnitPrice:       pgNumericToPtrFloat(a.UnitPrice),
		Presentation:    a.Presentation,
		TrackByLot:      a.TrackByLot,
		TrackBySerial:   a.TrackBySerial,
		TrackExpiration: a.TrackExpiration,
		MinQuantity:     pgInt4ToPtrInt(a.MinQuantity),
		MaxQuantity:     pgInt4ToPtrInt(a.MaxQuantity),
		ImageURL:        pgTextToPtrString(a.ImageUrl),
		IsActive:        pgBoolToPtrBool(a.IsActive),
		CreatedAt:       pgTimestampToTime(a.CreatedAt),
		UpdatedAt:       pgTimestampToTime(a.UpdatedAt),
	}
}

func sqlcLotToDatabase(l sqlc.Lot) database.Lot {
	st := l.Status
	return database.Lot{
		ID:             int(l.ID),
		LotNumber:      l.LotNumber,
		SKU:            l.Sku,
		Quantity:       pgNumericToFloat(l.Quantity),
		ExpirationDate: pgTimestampToPtrTime(l.ExpirationDate),
		CreatedAt:      pgTimestampToTime(l.CreatedAt),
		UpdatedAt:      pgTimestampToTime(l.UpdatedAt),
		Status:         &st,
	}
}

func sqlcSerialToDatabase(s sqlc.Serial) database.Serial {
	return database.Serial{
		ID:           int(s.ID),
		SerialNumber: s.SerialNumber,
		SKU:          s.Sku,
		Status:       s.Status,
		CreatedAt:    pgTimestampToTime(s.CreatedAt),
		UpdatedAt:    pgTimestampToTime(s.UpdatedAt),
	}
}

// pgtype conversion helpers
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func ptrFloatToPgNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(*f, 'f', -1, 64))
	return n
}

func ptrIntToPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

func pgTextToPtrString(t pgtype.Text) *string {
	if !t.Valid || t.String == "" {
		return nil
	}
	return &t.String
}

func pgNumericToPtrFloat(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

func pgNumericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func pgInt4ToPtrInt(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

func pgBoolToPtrBool(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}

func pgTimestampToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return tools.GetCurrentTime()
	}
	return t.Time
}

func pgTimestampToPtrTime(t pgtype.Timestamp) *time.Time {
	if !t.Valid || t.Time.IsZero() {
		return nil
	}
	return &t.Time
}

