package services

import (
	"errors"
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type StockTransfersService struct {
	Repository       ports.StockTransfersRepository
	LocationsRepository ports.LocationsRepository
	DB               *gorm.DB
}

func NewStockTransfersService(repo ports.StockTransfersRepository) *StockTransfersService {
	return &StockTransfersService{Repository: repo}
}

// NewStockTransfersServiceWithExecute builds the service with ExecuteTransfer support (locations + GORM DB).
func NewStockTransfersServiceWithExecute(repo ports.StockTransfersRepository, locationsRepo ports.LocationsRepository, db *gorm.DB) *StockTransfersService {
	return &StockTransfersService{
		Repository:         repo,
		LocationsRepository: locationsRepo,
		DB:                 db,
	}
}

func (s *StockTransfersService) ListStockTransfers(status string) ([]database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.ListStockTransfers(status)
}

func (s *StockTransfersService) GetStockTransferByID(id string) (*database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.GetStockTransferByID(id)
}

func (s *StockTransfersService) GetStockTransferByTransferNumber(transferNumber string) (*database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.GetStockTransferByTransferNumber(transferNumber)
}

func (s *StockTransfersService) CreateStockTransfer(req *requests.StockTransferCreate, createdBy string) (*database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.CreateStockTransfer(req, createdBy)
}

func (s *StockTransfersService) UpdateStockTransfer(id string, req *requests.StockTransferUpdate) (*database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.UpdateStockTransfer(id, req)
}

func (s *StockTransfersService) UpdateStockTransferStatus(id string, status string) (*database.StockTransfer, *responses.InternalResponse) {
	return s.Repository.UpdateStockTransferStatus(id, status)
}

func (s *StockTransfersService) DeleteStockTransfer(id string) *responses.InternalResponse {
	return s.Repository.DeleteStockTransfer(id)
}

func (s *StockTransfersService) ListStockTransferLines(transferID string) ([]database.StockTransferLine, *responses.InternalResponse) {
	return s.Repository.ListStockTransferLines(transferID)
}

func (s *StockTransfersService) CreateStockTransferLine(transferID string, req *requests.StockTransferLineInput) (*database.StockTransferLine, *responses.InternalResponse) {
	return s.Repository.CreateStockTransferLine(transferID, req)
}

func (s *StockTransfersService) UpdateStockTransferLine(lineID string, req *requests.StockTransferLineUpdate) (*database.StockTransferLine, *responses.InternalResponse) {
	return s.Repository.UpdateStockTransferLine(lineID, req)
}

func (s *StockTransfersService) DeleteStockTransferLine(lineID string) *responses.InternalResponse {
	return s.Repository.DeleteStockTransferLine(lineID)
}

// ExecuteTransfer moves stock from source to destination: decrements inventory at from_location,
// increments at to_location, creates outbound/inbound movements, and sets transfer status to completed.
// Requires LocationsRepository and DB to be set (use NewStockTransfersServiceWithExecute).
func (s *StockTransfersService) ExecuteTransfer(transferID, userID string) (*database.StockTransfer, *responses.InternalResponse) {
	if s.LocationsRepository == nil || s.DB == nil {
		return nil, &responses.InternalResponse{
			Message:    "Execute transfer is not configured (missing locations or database)",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		}
	}

	transfer, resp := s.Repository.GetStockTransferByID(transferID)
	if resp != nil {
		return nil, resp
	}
	if transfer == nil {
		return nil, &responses.InternalResponse{
			Message:    "Stock transfer not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if transfer.Status != "draft" && transfer.Status != "in_progress" {
		return nil, &responses.InternalResponse{
			Message:    fmt.Sprintf("Transfer cannot be executed in status %q", transfer.Status),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	lines, resp := s.Repository.ListStockTransferLines(transferID)
	if resp != nil {
		return nil, resp
	}
	if len(lines) == 0 {
		return nil, &responses.InternalResponse{
			Message:    "Transfer has no lines",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	fromLoc, resp := s.LocationsRepository.GetLocationByID(transfer.FromLocationID)
	if resp != nil || fromLoc == nil {
		if resp != nil {
			return nil, resp
		}
		return nil, &responses.InternalResponse{
			Message:    "From location not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	toLoc, resp := s.LocationsRepository.GetLocationByID(transfer.ToLocationID)
	if resp != nil || toLoc == nil {
		if resp != nil {
			return nil, resp
		}
		return nil, &responses.InternalResponse{
			Message:    "To location not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}

	fromCode := fromLoc.LocationCode
	toCode := toLoc.LocationCode

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range lines {
			sku := line.Sku
			qty := line.Quantity

			var fromInv database.Inventory
			// Lock the inventory row to prevent race conditions during concurrent transfers
			// or simultaneous picking operations (B3e A5).
			if err := tx.Raw(
				`SELECT id, sku, location, quantity, reserved_qty FROM inventory WHERE sku = ? AND location = ? FOR UPDATE`,
				sku, fromCode,
			).Scan(&fromInv).Error; err != nil {
				return fmt.Errorf("find inventory %s at %s: %w", sku, fromCode, err)
			}
			if fromInv.ID == "" {
				return fmt.Errorf("insufficient stock: SKU %s not found at source location %s", sku, fromCode)
			}
			// B3e (A5): check available (non-reserved) stock.
			available := fromInv.Quantity - fromInv.ReservedQty
			if qty > available {
				return fmt.Errorf(
					"no puede transferir %.2f de %s en %s — hay %.2f reservadas en pickings activos (disponible: %.2f)",
					qty, sku, fromCode, fromInv.ReservedQty, available,
				)
			}
			if fromInv.Quantity < qty {
				return fmt.Errorf("insufficient stock: SKU %s at %s has %.3f, need %.3f", sku, fromCode, fromInv.Quantity, qty)
			}

			newFromQty := fromInv.Quantity - qty
			if err := tx.Model(&database.Inventory{}).Where("id = ?", fromInv.ID).Update("quantity", newFromQty).Error; err != nil {
				return fmt.Errorf("update inventory %s at source: %w", sku, err)
			}

			movOutID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate movement id: %w", err)
			}
			movOut := &database.InventoryMovement{
				ID:             movOutID,
				SKU:            sku,
				Location:       fromCode,
				MovementType:   "outbound",
				Quantity:       -qty,
				RemainingStock: newFromQty,
				Reason:         strPtr("stock transfer " + transfer.TransferNumber),
				CreatedBy:      userID,
				CreatedAt:      tools.GetCurrentTime(),
			}
			if err := tx.Create(movOut).Error; err != nil {
				return fmt.Errorf("create outbound movement: %w", err)
			}

			var toInv database.Inventory
			errFind := tx.Where("sku = ? AND location = ?", sku, toCode).First(&toInv).Error
			if errFind != nil {
				if errors.Is(errFind, gorm.ErrRecordNotFound) {
					var article database.Article
					if err := tx.Where("sku = ?", sku).First(&article).Error; err != nil {
						return fmt.Errorf("article %s not found: %w", sku, err)
					}
					invID, err := tools.GenerateNanoid(tx)
					if err != nil {
						return fmt.Errorf("generate inventory id: %w", err)
					}
					pres := article.Presentation
					if line.Presentation != nil && *line.Presentation != "" {
						pres = *line.Presentation
					}
					toInv = database.Inventory{
						ID:           invID,
						SKU:          sku,
						Name:         article.Name,
						Location:     toCode,
						Quantity:     qty,
						Status:       "available",
						Presentation: pres,
						CreatedAt:    tools.GetCurrentTime(),
						UpdatedAt:    tools.GetCurrentTime(),
					}
					if err := tx.Create(&toInv).Error; err != nil {
						return fmt.Errorf("create inventory at destination: %w", err)
					}
				} else {
					return fmt.Errorf("find inventory %s at destination: %w", sku, errFind)
				}
			} else {
				toInv.Quantity += qty
				toInv.UpdatedAt = tools.GetCurrentTime()
				if err := tx.Save(&toInv).Error; err != nil {
					return fmt.Errorf("update inventory %s at destination: %w", sku, err)
				}
			}

			movInID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate movement id: %w", err)
			}
			movIn := &database.InventoryMovement{
				ID:             movInID,
				SKU:            sku,
				Location:       toCode,
				MovementType:   "inbound",
				Quantity:       qty,
				RemainingStock: toInv.Quantity,
				Reason:         strPtr("stock transfer " + transfer.TransferNumber),
				CreatedBy:      userID,
				CreatedAt:      tools.GetCurrentTime(),
			}
			if err := tx.Create(movIn).Error; err != nil {
				return fmt.Errorf("create inbound movement: %w", err)
			}
		}

		if err := tx.Exec("UPDATE stock_transfers SET status = 'completed', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", transferID).Error; err != nil {
			return fmt.Errorf("update transfer status: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: err.Error(),
			Handled: true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	updated, _ := s.Repository.GetStockTransferByID(transferID)
	return updated, nil
}

func strPtr(s string) *string {
	return &s
}
