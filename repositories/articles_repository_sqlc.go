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
		out[i] = articleRowToDatabase(articleRowData{
			ID: a.ID, Sku: a.Sku, Name: a.Name, Description: a.Description, UnitPrice: a.UnitPrice,
			Presentation: a.Presentation, TrackByLot: a.TrackByLot, TrackBySerial: a.TrackBySerial,
			TrackExpiration: a.TrackExpiration, RotationStrategy: a.RotationStrategy,
			MinQuantity: a.MinQuantity, MaxQuantity: a.MaxQuantity, ImageUrl: a.ImageUrl,
			IsActive: a.IsActive, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
			CategoryID: a.CategoryID, ShelfLifeInDays: a.ShelfLifeInDays, SafetyStock: a.SafetyStock,
			BatchNumberSeries: a.BatchNumberSeries, SerialNumberSeries: a.SerialNumberSeries,
			MinOrderQty: a.MinOrderQty, DefaultLocationID: a.DefaultLocationID,
			ReceivingNotes: a.ReceivingNotes, ShippingNotes: a.ShippingNotes,
		})
	}
	return out, nil
}

func (r *ArticlesRepositorySQLC) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	a, err := r.queries.GetArticleByID(ctx, id)
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
	art := articleRowToDatabase(articleRowData{
		ID: a.ID, Sku: a.Sku, Name: a.Name, Description: a.Description, UnitPrice: a.UnitPrice,
		Presentation: a.Presentation, TrackByLot: a.TrackByLot, TrackBySerial: a.TrackBySerial,
		TrackExpiration: a.TrackExpiration, RotationStrategy: a.RotationStrategy,
		MinQuantity: a.MinQuantity, MaxQuantity: a.MaxQuantity, ImageUrl: a.ImageUrl,
		IsActive: a.IsActive, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		CategoryID: a.CategoryID, ShelfLifeInDays: a.ShelfLifeInDays, SafetyStock: a.SafetyStock,
		BatchNumberSeries: a.BatchNumberSeries, SerialNumberSeries: a.SerialNumberSeries,
		MinOrderQty: a.MinOrderQty, DefaultLocationID: a.DefaultLocationID,
		ReceivingNotes: a.ReceivingNotes, ShippingNotes: a.ShippingNotes,
	})
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
	art := articleRowToDatabase(articleRowData{
		ID: a.ID, Sku: a.Sku, Name: a.Name, Description: a.Description, UnitPrice: a.UnitPrice,
		Presentation: a.Presentation, TrackByLot: a.TrackByLot, TrackBySerial: a.TrackBySerial,
		TrackExpiration: a.TrackExpiration, RotationStrategy: a.RotationStrategy,
		MinQuantity: a.MinQuantity, MaxQuantity: a.MaxQuantity, ImageUrl: a.ImageUrl,
		IsActive: a.IsActive, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		CategoryID: a.CategoryID, ShelfLifeInDays: a.ShelfLifeInDays, SafetyStock: a.SafetyStock,
		BatchNumberSeries: a.BatchNumberSeries, SerialNumberSeries: a.SerialNumberSeries,
		MinOrderQty: a.MinOrderQty, DefaultLocationID: a.DefaultLocationID,
		ReceivingNotes: a.ReceivingNotes, ShippingNotes: a.ShippingNotes,
	})
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

	rotationStrategy := strings.TrimSpace(strings.ToLower(data.RotationStrategy))
	if rotationStrategy == "" || (rotationStrategy != "fifo" && rotationStrategy != "fefo") {
		rotationStrategy = "fifo"
	}

	arg := sqlc.CreateArticleParams{
		Sku:                data.SKU,
		Name:               data.Name,
		Description:        ptrStringToPgText(data.Description),
		UnitPrice:          ptrFloatToPgNumeric(data.UnitPrice),
		Presentation:       data.Presentation,
		TrackByLot:         data.TrackByLot,
		TrackBySerial:      data.TrackBySerial,
		TrackExpiration:    data.TrackExpiration,
		RotationStrategy:   rotationStrategy,
		MinQuantity:        ptrIntToPgInt4(data.MinQuantity),
		MaxQuantity:        ptrIntToPgInt4(data.MaxQuantity),
		ImageUrl:           ptrStringToPgText(data.ImageURL),
		CategoryID:         ptrStringToPgText(data.CategoryID),
		ShelfLifeInDays:    ptrIntToPgInt4(data.ShelfLifeInDays),
		SafetyStock:        floatToPgNumeric(data.SafetyStock),
		BatchNumberSeries:  ptrStringToPgText(data.BatchNumberSeries),
		SerialNumberSeries: ptrStringToPgText(data.SerialNumberSeries),
		MinOrderQty:        floatToPgNumeric(data.MinOrderQty),
		DefaultLocationID:  ptrStringToPgText(data.DefaultLocationID),
		ReceivingNotes:     ptrStringToPgText(data.ReceivingNotes),
		ShippingNotes:      ptrStringToPgText(data.ShippingNotes),
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

func (r *ArticlesRepositorySQLC) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	ctx := context.Background()
	existing, err := r.queries.GetArticleByID(ctx, id)
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

	rotationStrategy := strings.TrimSpace(strings.ToLower(data.RotationStrategy))
	if rotationStrategy == "" || (rotationStrategy != "fifo" && rotationStrategy != "fefo") {
		rotationStrategy = existing.RotationStrategy
	}

	arg := sqlc.UpdateArticleParams{
		ID:                 existing.ID,
		Sku:                data.SKU,
		Name:               data.Name,
		Description:        ptrStringToPgText(data.Description),
		UnitPrice:          ptrFloatToPgNumeric(data.UnitPrice),
		Presentation:       data.Presentation,
		TrackByLot:         data.TrackByLot,
		TrackBySerial:      data.TrackBySerial,
		TrackExpiration:    data.TrackExpiration,
		RotationStrategy:   rotationStrategy,
		MinQuantity:        ptrIntToPgInt4(data.MinQuantity),
		MaxQuantity:        ptrIntToPgInt4(data.MaxQuantity),
		ImageUrl:           ptrStringToPgText(data.ImageURL),
		IsActive:           existing.IsActive,
		CategoryID:         ptrStringToPgText(data.CategoryID),
		ShelfLifeInDays:    ptrIntToPgInt4(data.ShelfLifeInDays),
		SafetyStock:        floatToPgNumeric(data.SafetyStock),
		BatchNumberSeries:  ptrStringToPgText(data.BatchNumberSeries),
		SerialNumberSeries: ptrStringToPgText(data.SerialNumberSeries),
		MinOrderQty:        floatToPgNumeric(data.MinOrderQty),
		DefaultLocationID:  ptrStringToPgText(data.DefaultLocationID),
		ReceivingNotes:     ptrStringToPgText(data.ReceivingNotes),
		ShippingNotes:      ptrStringToPgText(data.ShippingNotes),
	}

	updated, err := r.queries.UpdateArticle(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al actualizar el artículo", Handled: false}
	}
	art := articleRowToDatabase(articleRowData{
		ID: updated.ID, Sku: updated.Sku, Name: updated.Name, Description: updated.Description, UnitPrice: updated.UnitPrice,
		Presentation: updated.Presentation, TrackByLot: updated.TrackByLot, TrackBySerial: updated.TrackBySerial,
		TrackExpiration: updated.TrackExpiration, RotationStrategy: updated.RotationStrategy,
		MinQuantity: updated.MinQuantity, MaxQuantity: updated.MaxQuantity, ImageUrl: updated.ImageUrl,
		IsActive: updated.IsActive, CreatedAt: updated.CreatedAt, UpdatedAt: updated.UpdatedAt,
		CategoryID: updated.CategoryID, ShelfLifeInDays: updated.ShelfLifeInDays, SafetyStock: updated.SafetyStock,
		BatchNumberSeries: updated.BatchNumberSeries, SerialNumberSeries: updated.SerialNumberSeries,
		MinOrderQty: updated.MinOrderQty, DefaultLocationID: updated.DefaultLocationID,
		ReceivingNotes: updated.ReceivingNotes, ShippingNotes: updated.ShippingNotes,
	})
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

func (r *ArticlesRepositorySQLC) DeleteArticle(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeleteArticle(ctx, id)
	if err != nil {
		tools.LogRepoError("articles", "DeleteArticle", err, "Error al eliminar el artículo")
		return &responses.InternalResponse{Error: err, Message: "Error al eliminar el artículo", Handled: false}
	}
	return nil
}

func (r *ArticlesRepositorySQLC) ImportArticlesFromExcel(fileBytes []byte) ([]string, []string, []*responses.InternalResponse) {
	imported, skipped, errs := []string{}, []string{}, []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return imported, skipped, append(errs, &responses.InternalResponse{Error: err, Message: "Error al abrir el archivo", Handled: false})
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return imported, skipped, append(errs, &responses.InternalResponse{Message: "Sin hojas de datos", Handled: true})
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return imported, skipped, append(errs, &responses.InternalResponse{Error: err, Message: "Error al leer filas", Handled: false})
	}

	for i, row := range rows {
		if i < 8 || len(row) < 10 {
			continue
		}
		sku := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		if sku == "" || name == "" {
			continue
		}
		if strings.EqualFold(sku, "ART-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: ejemplo omitido", i+1))
			continue
		}
		presentation := strings.TrimSpace(row[4])
		if presentation == "" {
			errs = append(errs, &responses.InternalResponse{Message: fmt.Sprintf("Fila %d: presentación requerida", i+1), Handled: true})
			continue
		}
		rowReq := requests.ArticleImportRow{
			SKU: sku, Name: name, Description: strings.TrimSpace(row[2]),
			UnitPrice: strings.TrimSpace(row[3]), Presentation: presentation,
			TrackByLot: strings.TrimSpace(row[5]), TrackBySerial: strings.TrimSpace(row[6]),
			TrackExpiration: strings.TrimSpace(row[7]), MaxQuantity: strings.TrimSpace(row[8]),
			MinQuantity: strings.TrimSpace(row[9]),
		}
		if len(row) > 10 {
			rowReq.RotationStrategy = strings.TrimSpace(row[10])
		}
		imp, sk, rowErrs := r.ImportArticlesFromJSON([]requests.ArticleImportRow{rowReq})
		imported = append(imported, imp...)
		skipped = append(skipped, sk...)
		errs = append(errs, rowErrs...)
	}
	return imported, skipped, errs
}

func (r *ArticlesRepositorySQLC) ImportArticlesFromJSON(rows []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	imported, skipped, errs := []string{}, []string{}, []*responses.InternalResponse{}
	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		name := strings.TrimSpace(row.Name)
		if sku == "" || name == "" {
			skipped = append(skipped, fmt.Sprintf("Fila %d: SKU y nombre requeridos", i+1))
			continue
		}
		if strings.EqualFold(sku, "ART-001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: ejemplo omitido", i+1))
			continue
		}
		presentation := strings.TrimSpace(row.Presentation)
		if presentation == "" {
			errs = append(errs, &responses.InternalResponse{Message: fmt.Sprintf("Fila %d: presentación requerida", i+1), Handled: true})
			continue
		}
		rs := strings.ToLower(strings.TrimSpace(row.RotationStrategy))
		if rs != "fifo" && rs != "fefo" {
			rs = ""
		}
		var unitPrice *float64
		if p, e := strconv.ParseFloat(strings.TrimSpace(row.UnitPrice), 64); e == nil {
			unitPrice = &p
		}
		var minQty, maxQty *int
		if q, e := strconv.Atoi(strings.TrimSpace(row.MinQuantity)); e == nil {
			minQty = &q
		}
		if q, e := strconv.Atoi(strings.TrimSpace(row.MaxQuantity)); e == nil {
			maxQty = &q
		}
		desc := strings.TrimSpace(row.Description)
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}
		article := &requests.Article{
			SKU: sku, Name: name, Description: descPtr,
			UnitPrice: unitPrice, Presentation: presentation,
			TrackByLot: parseBoolCell(row.TrackByLot), TrackBySerial: parseBoolCell(row.TrackBySerial),
			TrackExpiration: parseBoolCell(row.TrackExpiration), RotationStrategy: rs,
			MinQuantity: minQty, MaxQuantity: maxQty,
		}
		resp := r.CreateArticle(article)
		if resp != nil {
			errs = append(errs, &responses.InternalResponse{Error: resp.Error, Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message), Handled: resp.Handled})
			continue
		}
		imported = append(imported, sku)
	}
	return imported, skipped, errs
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
		"Cantidad Máxima", "Cantidad Mínima", "Rotación (FIFO/FEFO)",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, article := range articles {
		row := idx + 7
		rotStr := article.RotationStrategy
		if rotStr == "" {
			rotStr = "fifo"
		}
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
			strings.ToUpper(rotStr),
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

// articleRowData holds the common shape of sqlc article row types (ListArticlesRow, GetArticleByIDRow, GetArticleBySkuRow, UpdateArticleRow).
type articleRowData struct {
	ID                 string
	Sku                string
	Name               string
	Description        pgtype.Text
	UnitPrice          pgtype.Numeric
	Presentation       string
	TrackByLot         bool
	TrackBySerial      bool
	TrackExpiration    bool
	RotationStrategy   string
	MinQuantity        pgtype.Int4
	MaxQuantity        pgtype.Int4
	ImageUrl           pgtype.Text
	IsActive           pgtype.Bool
	CreatedAt          pgtype.Timestamp
	UpdatedAt          pgtype.Timestamp
	CategoryID         pgtype.Text
	ShelfLifeInDays    pgtype.Int4
	SafetyStock        pgtype.Numeric
	BatchNumberSeries  pgtype.Text
	SerialNumberSeries pgtype.Text
	MinOrderQty        pgtype.Numeric
	DefaultLocationID  pgtype.Text
	ReceivingNotes     pgtype.Text
	ShippingNotes      pgtype.Text
}

func articleRowToDatabase(a articleRowData) database.Article {
	rotationStrategy := strings.TrimSpace(strings.ToLower(a.RotationStrategy))
	if rotationStrategy == "" {
		rotationStrategy = "fifo"
	}
	return database.Article{
		ID:                 a.ID,
		SKU:                a.Sku,
		Name:               a.Name,
		Description:        pgTextToPtrString(a.Description),
		UnitPrice:          pgNumericToPtrFloat(a.UnitPrice),
		Presentation:       a.Presentation,
		TrackByLot:         a.TrackByLot,
		TrackBySerial:      a.TrackBySerial,
		TrackExpiration:    a.TrackExpiration,
		RotationStrategy:   rotationStrategy,
		MinQuantity:        pgInt4ToPtrInt(a.MinQuantity),
		MaxQuantity:        pgInt4ToPtrInt(a.MaxQuantity),
		ImageURL:           pgTextToPtrString(a.ImageUrl),
		IsActive:           pgBoolToPtrBool(a.IsActive),
		CreatedAt:          pgTimestampToTime(a.CreatedAt),
		UpdatedAt:          pgTimestampToTime(a.UpdatedAt),
		CategoryID:         pgTextToPtrString(a.CategoryID),
		ShelfLifeInDays:    pgInt4ToPtrInt(a.ShelfLifeInDays),
		SafetyStock:        pgNumericToFloat(a.SafetyStock),
		BatchNumberSeries:  pgTextToPtrString(a.BatchNumberSeries),
		SerialNumberSeries: pgTextToPtrString(a.SerialNumberSeries),
		MinOrderQty:        pgNumericToFloat(a.MinOrderQty),
		DefaultLocationID:  pgTextToPtrString(a.DefaultLocationID),
		ReceivingNotes:     pgTextToPtrString(a.ReceivingNotes),
		ShippingNotes:      pgTextToPtrString(a.ShippingNotes),
	}
}

func sqlcLotToDatabase(l sqlc.Lot) database.Lot {
	st := l.Status
	return database.Lot{
		ID:             l.ID,
		LotNumber:      l.LotNumber,
		SKU:            l.Sku,
		Quantity:       pgNumericToFloat(l.Quantity),
		ExpirationDate: pgTimestampToPtrTime(l.ExpirationDate),
		CreatedAt:      pgTimestampToTime(l.CreatedAt),
		UpdatedAt:      pgTimestampToTime(l.UpdatedAt),
		Status:         &st,
		LotNotes:       pgTextToPtrString(l.LotNotes),
		ManufacturedAt: pgDateToPtrTime(l.ManufacturedAt),
		BestBeforeDate: pgDateToPtrTime(l.BestBeforeDate),
	}
}

func sqlcSerialToDatabase(s sqlc.Serial) database.Serial {
	return database.Serial{
		ID:           s.ID,
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

func pgDateToPtrTime(d pgtype.Date) *time.Time {
	if !d.Valid || d.Time.IsZero() {
		return nil
	}
	return &d.Time
}

func ptrStringToPgDate(s *string) pgtype.Date {
	if s == nil || *s == "" {
		return pgtype.Date{}
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}


func (r *ArticlesRepositorySQLC) ValidateImportRows(rows []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	results := make([]responses.ArticleValidationResult, 0, len(rows))
	seenSKUs := make(map[string]bool)

	// Load all articles once for in-memory similarity check
	allArticles, errResp := r.GetAllArticles()
	if errResp != nil {
		return nil, errResp
	}

	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		name := strings.TrimSpace(row.Name)
		result := responses.ArticleValidationResult{RowIndex: i, Row: row}

		// Field validation
		if sku == "" || name == "" || strings.TrimSpace(row.Presentation) == "" {
			result.Status = responses.ArticleStatusError
			result.FieldErrors = map[string]string{}
			if sku == "" { result.FieldErrors["sku"] = "SKU requerido" }
			if name == "" { result.FieldErrors["name"] = "Nombre requerido" }
			if strings.TrimSpace(row.Presentation) == "" { result.FieldErrors["presentation"] = "Presentación requerida" }
			results = append(results, result)
			continue
		}

		// Duplicate within batch
		skuKey := strings.ToLower(sku)
		if seenSKUs[skuKey] {
			result.Status = responses.ArticleStatusDuplicate
			results = append(results, result)
			continue
		}
		seenSKUs[skuKey] = true

		// Exact SKU match
		ctx := context.Background()
		existing, err := r.queries.GetArticleBySku(ctx, sku)
		if err == nil {
			isActive := false
			if existing.IsActive.Valid { isActive = existing.IsActive.Bool }
			result.Status = responses.ArticleStatusExists
			result.ExistingArticle = &responses.ArticleValidationMatch{
				ID: existing.ID, SKU: existing.Sku, Name: existing.Name,
				Presentation: existing.Presentation, IsActive: isActive,
			}
			results = append(results, result)
			continue
		}

		// In-memory similarity check
		keyword := strings.ToLower(name)
		if len(keyword) > 20 { keyword = keyword[:20] }
		var matches []responses.ArticleValidationMatch
		for _, a := range allArticles {
			if strings.ToLower(a.SKU) == strings.ToLower(sku) { continue }
			if strings.Contains(strings.ToLower(a.Name), keyword) {
				isActive := false
				if a.IsActive != nil { isActive = *a.IsActive }
				matches = append(matches, responses.ArticleValidationMatch{
					ID: a.ID, SKU: a.SKU, Name: a.Name,
					Presentation: a.Presentation, IsActive: isActive,
				})
				if len(matches) == 3 { break }
			}
		}
		if len(matches) > 0 {
			result.Status = responses.ArticleStatusSimilar
			result.SimilarArticles = matches
			results = append(results, result)
			continue
		}

		result.Status = responses.ArticleStatusNew
		results = append(results, result)
	}
	return results, nil
}

func (r *ArticlesRepositorySQLC) GenerateImportTemplate(language string) ([]byte, *responses.InternalResponse) {
	articles, errResp := r.GetAllArticles()
	if errResp != nil {
		return nil, errResp
	}
	var presentations []string
	for _, a := range articles {
		presentations = append(presentations, a.Presentation)
	}
	return buildImportTemplate(presentations, language)
}
