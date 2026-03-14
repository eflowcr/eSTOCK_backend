package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// StockTransfersRepository defines persistence for stock transfers and lines (WMS transfer orders).
type StockTransfersRepository interface {
	ListStockTransfers(status string) ([]database.StockTransfer, *responses.InternalResponse)
	GetStockTransferByID(id string) (*database.StockTransfer, *responses.InternalResponse)
	GetStockTransferByTransferNumber(transferNumber string) (*database.StockTransfer, *responses.InternalResponse)
	CreateStockTransfer(req *requests.StockTransferCreate, createdBy string) (*database.StockTransfer, *responses.InternalResponse)
	UpdateStockTransfer(id string, req *requests.StockTransferUpdate) (*database.StockTransfer, *responses.InternalResponse)
	UpdateStockTransferStatus(id string, status string) (*database.StockTransfer, *responses.InternalResponse)
	DeleteStockTransfer(id string) *responses.InternalResponse

	ListStockTransferLines(transferID string) ([]database.StockTransferLine, *responses.InternalResponse)
	CreateStockTransferLine(transferID string, req *requests.StockTransferLineInput) (*database.StockTransferLine, *responses.InternalResponse)
	UpdateStockTransferLine(lineID string, req *requests.StockTransferLineUpdate) (*database.StockTransferLine, *responses.InternalResponse)
	DeleteStockTransferLine(lineID string) *responses.InternalResponse
}
