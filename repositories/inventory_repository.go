package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	DB *gorm.DB
}

func (r *InventoryRepository) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	var items []database.Inventory
	err := r.DB.
		Order("sku ASC").
		Find(&items).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory",
			Handled: false,
		}
	}

	if len(items) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No inventory found",
			Handled: true,
		}
	}

	var enhanced []*dto.EnhancedInventory

	for _, item := range items {
		// Obtener información del artículo
		var article database.Article
		err := r.DB.Where("sku = ?", item.SKU).First(&article).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Error:   err,
				Message: "Failed to fetch article for inventory item",
				Handled: false,
			}
		}

		// Obtener lotes si aplica
		var lots []database.Lot
		if article.TrackByLot {
			// lots, _ = r.GetLotsByInventoryID(item.ID)
		}

		// Obtener seriales si aplica
		var serials []database.Serial
		if article.TrackBySerial {
			// serials, _ = r.GetSerialsByInventoryID(item.ID)
		}

		enhanced = append(enhanced, &dto.EnhancedInventory{
			ID:              item.ID,
			SKU:             item.SKU,
			Location:        item.Location,
			Quantity:        item.Quantity,
			Status:          item.Status,
			UnitPrice:       *item.UnitPrice,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
			Name:            article.Name,
			Description:     *article.Description,
			Presentation:    article.Presentation,
			TrackByLot:      article.TrackByLot,
			TrackBySerial:   article.TrackBySerial,
			TrackExpiration: article.TrackExpiration,
			ImageURL:        *article.ImageURL,
			MinQuantity:     *article.MinQuantity,
			MaxQuantity:     *article.MaxQuantity,
			Lots:            lots,
			Serials:         serials,
		})
	}

	return enhanced, nil
}

func (r *InventoryRepository) CreateInventory(item *requests.CreateInventory) *responses.InternalResponse {
	// 1 - Check if sku exists in the location
	var inventoryCount int64
	err := r.DB.Model(&database.Inventory{}).
		Where("sku = ? AND location = ?", item.SKU, item.Location).
		Count(&inventoryCount).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check existing inventory",
			Handled: false,
		}
	}

	if inventoryCount > 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Inventory with this SKU already exists in the specified location",
			Handled: true,
		}
	}

	// 2 - Get article information
	var article database.Article
	err = r.DB.Where("sku = ?", item.SKU).First(&article).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article for inventory creation",
			Handled: false,
		}
	}

	if article.ID == 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Article not found for the provided SKU",
			Handled: true,
		}
	}

	var inventory database.Inventory

	inventory.SKU = item.SKU
	inventory.Name = item.Name
	inventory.Description = item.Description
	inventory.Location = item.Location
	inventory.Quantity = item.Quantity
	inventory.Status = "available"
	inventory.Presentation = article.Presentation
	inventory.UnitPrice = item.UnitPrice
	inventory.CreatedAt = tools.GetCurrentTime()
	inventory.UpdatedAt = tools.GetCurrentTime()

	if item.Name != "" {
		inventory.Name = item.Name
	}

	if item.Description != nil {
		inventory.Description = item.Description
	}

	if item.UnitPrice != nil {
		inventory.UnitPrice = item.UnitPrice
	}

	if err := r.DB.Create(&inventory).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create inventory",
			Handled: false,
		}
	}

	// 3 - Create lots if applicable
	if article.TrackByLot && item.Lots != nil {
		for i := 0; i < len(item.Lots); i++ {
			var lotCount int64

			err := r.DB.Model(&database.Lot{}).
				Where("lot_number = ? AND sku = ?", item.Lots[i].LotNumber, item.SKU).
				Count(&lotCount).Error

			if err != nil {
				return &responses.InternalResponse{
					Error:   err,
					Message: "Failed to check existing lot",
					Handled: false,
				}
			}

			if lotCount == 0 {
				// Create new lot
				lot := &database.Lot{
					LotNumber:      item.Lots[i].LotNumber,
					SKU:            item.SKU,
					Quantity:       item.Lots[i].Quantity,
					ExpirationDate: item.Lots[i].ExpirationDate,
					CreatedAt:      tools.GetCurrentTime(),
					UpdatedAt:      tools.GetCurrentTime(),
				}

				if err := r.DB.Create(lot).Error; err != nil {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to create lot",
						Handled: false,
					}
				}

				// Create inventory_lot association
				inventoryLot := &database.InventoryLot{
					InventoryID: inventory.ID,
					LotID:       lot.ID,
					Quantity:    item.Lots[i].Quantity,
					Location:    item.Location,
				}

				if err := r.DB.Create(inventoryLot).Error; err != nil {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to create inventory_lot association",
						Handled: false,
					}
				}
			}
		}
	}

	// 4 - Create serials if applicable
	if article.TrackBySerial && item.Serials != nil {
		for i := 0; i < len(item.Serials); i++ {
			// Check if serial already exists
			var serialCount int64
			err := r.DB.Model(&database.Serial{}).
				Where("serial_number = ? AND sku = ?", item.Serials[i].SerialNumber, item.SKU).
				Count(&serialCount).Error

			if err != nil {
				return &responses.InternalResponse{
					Error:   err,
					Message: "Failed to check existing serial",
					Handled: false,
				}
			}

			if serialCount == 0 {
				// Create new serial
				newSerial := &database.Serial{
					SerialNumber: item.Serials[i].SerialNumber,
					SKU:          item.SKU,
					CreatedAt:    tools.GetCurrentTime(),
					UpdatedAt:    tools.GetCurrentTime(),
					Status:       "available",
				}

				if err := r.DB.Create(newSerial).Error; err != nil {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to create serial",
						Handled: false,
					}
				}
			}

			// Create inventory_serial association
			inventorySerial := &database.InventorySerial{
				InventoryID: inventory.ID,
				SerialID:    item.Serials[i].ID,
				Location:    item.Location,
			}

			if err := r.DB.Create(inventorySerial).Error; err != nil {
				return &responses.InternalResponse{
					Error:   err,
					Message: "Failed to create inventory_serial association",
					Handled: false,
				}
			}
		}
	}

	return nil
}
