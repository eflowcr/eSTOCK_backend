package repositories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type StockTransfersRepositorySQLC struct {
	queries *sqlc.Queries
}

func NewStockTransfersRepositorySQLC(queries *sqlc.Queries) *StockTransfersRepositorySQLC {
	return &StockTransfersRepositorySQLC{queries: queries}
}

var _ ports.StockTransfersRepository = (*StockTransfersRepositorySQLC)(nil)

func (r *StockTransfersRepositorySQLC) ListStockTransfers(status string) ([]database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	var list []sqlc.StockTransfer
	var err error
	if status != "" {
		list, err = r.queries.ListStockTransfersByStatus(ctx, status)
	} else {
		list, err = r.queries.ListStockTransfers(ctx)
	}
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing stock transfers", Handled: false}
	}
	out := make([]database.StockTransfer, len(list))
	for i, row := range list {
		out[i] = sqlcTransferToDatabase(row)
	}
	return out, nil
}

func (r *StockTransfersRepositorySQLC) GetStockTransferByID(id string) (*database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetStockTransferByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Stock transfer not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting stock transfer", Handled: false}
	}
	t := sqlcTransferToDatabase(row)
	return &t, nil
}

func (r *StockTransfersRepositorySQLC) GetStockTransferByTransferNumber(transferNumber string) (*database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.GetStockTransferByTransferNumber(ctx, transferNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Stock transfer not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting stock transfer", Handled: false}
	}
	t := sqlcTransferToDatabase(row)
	return &t, nil
}

func (r *StockTransfersRepositorySQLC) CreateStockTransfer(req *requests.StockTransferCreate, createdBy string) (*database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	if req.FromLocationID == req.ToLocationID {
		return nil, &responses.InternalResponse{Message: "From and to location must be different", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	if len(req.Lines) == 0 {
		return nil, &responses.InternalResponse{Message: "At least one line is required", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	transferNumber := generateTransferNumber()
	arg := sqlc.CreateStockTransferParams{
		TransferNumber: transferNumber,
		FromLocationID: req.FromLocationID,
		ToLocationID:   req.ToLocationID,
		Status:         "draft",
		CreatedBy:      createdBy,
		AssignedTo:     textToPgType(req.AssignedTo),
		Notes:          textToPgType(req.Notes),
	}
	row, err := r.queries.CreateStockTransfer(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating stock transfer", Handled: false}
	}
	transfer := sqlcTransferToDatabase(row)

	for _, line := range req.Lines {
		lineArg := sqlc.CreateStockTransferLineParams{
			StockTransferID: transfer.ID,
			Sku:             line.Sku,
			Quantity:        floatToPgNumericStockTransfer(line.Quantity),
			Presentation:    textToPgType(line.Presentation),
			LineStatus:      "pending",
		}
		_, err = r.queries.CreateStockTransferLine(ctx, lineArg)
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error creating stock transfer line", Handled: false}
		}
	}

	return &transfer, nil
}

func (r *StockTransfersRepositorySQLC) UpdateStockTransfer(id string, req *requests.StockTransferUpdate) (*database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	_, err := r.queries.GetStockTransferByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Stock transfer not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error getting stock transfer", Handled: false}
	}
	if req.FromLocationID == req.ToLocationID {
		return nil, &responses.InternalResponse{Message: "From and to location must be different", Handled: true, StatusCode: responses.StatusBadRequest}
	}

	arg := sqlc.UpdateStockTransferParams{
		ID:             id,
		FromLocationID: req.FromLocationID,
		ToLocationID:   req.ToLocationID,
		Status:         req.Status,
		AssignedTo:     textToPgType(req.AssignedTo),
		Notes:          textToPgType(req.Notes),
	}
	row, err := r.queries.UpdateStockTransfer(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating stock transfer", Handled: false}
	}
	t := sqlcTransferToDatabase(row)
	return &t, nil
}

func (r *StockTransfersRepositorySQLC) UpdateStockTransferStatus(id string, status string) (*database.StockTransfer, *responses.InternalResponse) {
	ctx := context.Background()
	row, err := r.queries.UpdateStockTransferStatus(ctx, sqlc.UpdateStockTransferStatusParams{ID: id, Status: status})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Stock transfer not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating stock transfer status", Handled: false}
	}
	t := sqlcTransferToDatabase(row)
	return &t, nil
}

func (r *StockTransfersRepositorySQLC) DeleteStockTransfer(id string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeleteStockTransfer(ctx, id)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting stock transfer", Handled: false}
	}
	return nil
}

func (r *StockTransfersRepositorySQLC) ListStockTransferLines(transferID string) ([]database.StockTransferLine, *responses.InternalResponse) {
	ctx := context.Background()
	list, err := r.queries.ListStockTransferLinesByTransferID(ctx, transferID)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error listing stock transfer lines", Handled: false}
	}
	out := make([]database.StockTransferLine, len(list))
	for i, row := range list {
		out[i] = sqlcTransferLineToDatabase(row)
	}
	return out, nil
}

func (r *StockTransfersRepositorySQLC) CreateStockTransferLine(transferID string, req *requests.StockTransferLineInput) (*database.StockTransferLine, *responses.InternalResponse) {
	ctx := context.Background()
	arg := sqlc.CreateStockTransferLineParams{
		StockTransferID: transferID,
		Sku:             req.Sku,
		Quantity:        floatToPgNumericStockTransfer(req.Quantity),
		Presentation:    textToPgType(req.Presentation),
		LineStatus:      "pending",
	}
	row, err := r.queries.CreateStockTransferLine(ctx, arg)
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error creating stock transfer line", Handled: false}
	}
	line := sqlcTransferLineToDatabase(row)
	return &line, nil
}

func (r *StockTransfersRepositorySQLC) UpdateStockTransferLine(lineID string, req *requests.StockTransferLineUpdate) (*database.StockTransferLine, *responses.InternalResponse) {
	ctx := context.Background()
	arg := sqlc.UpdateStockTransferLineParams{
		ID:           lineID,
		Quantity:     floatToPgNumericStockTransfer(req.Quantity),
		Presentation: textToPgType(req.Presentation),
		LineStatus:   req.LineStatus,
	}
	if arg.LineStatus == "" {
		arg.LineStatus = "pending"
	}
	row, err := r.queries.UpdateStockTransferLine(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &responses.InternalResponse{Message: "Stock transfer line not found", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error updating stock transfer line", Handled: false}
	}
	line := sqlcTransferLineToDatabase(row)
	return &line, nil
}

func (r *StockTransfersRepositorySQLC) DeleteStockTransferLine(lineID string) *responses.InternalResponse {
	ctx := context.Background()
	err := r.queries.DeleteStockTransferLine(ctx, lineID)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error deleting stock transfer line", Handled: false}
	}
	return nil
}

func textToPgType(s *string) pgtype.Text {
	var t pgtype.Text
	if s != nil {
		t.Valid = true
		t.String = *s
	}
	return t
}

func sqlcTransferToDatabase(row sqlc.StockTransfer) database.StockTransfer {
	return database.StockTransfer{
		ID:             row.ID,
		TransferNumber: row.TransferNumber,
		FromLocationID: row.FromLocationID,
		ToLocationID:   row.ToLocationID,
		Status:         row.Status,
		CreatedBy:      row.CreatedBy,
		AssignedTo:     pgTextToPtrString(row.AssignedTo),
		Notes:          pgTextToPtrString(row.Notes),
		CreatedAt:      pgTimestampToTime(row.CreatedAt),
		UpdatedAt:      pgTimestampToTime(row.UpdatedAt),
		CompletedAt:    pgTimestampToPtrTime(row.CompletedAt),
	}
}

func sqlcTransferLineToDatabase(row sqlc.StockTransferLine) database.StockTransferLine {
	return database.StockTransferLine{
		ID:              row.ID,
		StockTransferID: row.StockTransferID,
		Sku:             row.Sku,
		Quantity:        pgNumericToFloat(row.Quantity),
		Presentation:   pgTextToPtrString(row.Presentation),
		LineStatus:      row.LineStatus,
		CreatedAt:       pgTimestampToTime(row.CreatedAt),
	}
}

func floatToPgNumericStockTransfer(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

func generateTransferNumber() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("TRF-%s-%s", time.Now().Format("20060102150405"), hex.EncodeToString(b))
}
