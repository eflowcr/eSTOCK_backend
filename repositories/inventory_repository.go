package repositories

import (
	"errors"
	"fmt"

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
			// Get lots associated with the inventory item
			err = r.DB.
				Table(database.Lot{}.TableName()).
				Joins("JOIN inventory_lots ON lots.id = inventory_lots.lot_id").
				Where("inventory_lots.inventory_id = ?", item.ID).
				Find(&lots).Error

			if err != nil {
				return nil, &responses.InternalResponse{
					Error:   err,
					Message: "Failed to fetch lots for inventory item",
					Handled: false,
				}
			}
		}

		// Obtener seriales si aplica
		var serials []database.Serial
		if article.TrackBySerial {
			// Get serials associated with the inventory item
			err = r.DB.
				Table(database.Serial{}.TableName()).
				Joins("JOIN inventory_serials ON serials.id = inventory_serials.serial_id").
				Where("inventory_serials.inventory_id = ?", item.ID).
				Find(&serials).Error

			if err != nil {
				return nil, &responses.InternalResponse{
					Error:   err,
					Message: "Failed to fetch serials for inventory item",
					Handled: false,
				}
			}
		}

		// Image URL
		imageURL := ""
		if article.ImageURL != nil {
			imageURL = *article.ImageURL
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
			ImageURL:        imageURL,
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

func (r *InventoryRepository) UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse {
	// 1 - Get the current inventory item
	var inventory database.Inventory
	err := r.DB.Where("sku = ? AND location = ?", item.SKU, item.Location).First(&inventory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{
				Error:   nil,
				Message: "Inventory item not found",
				Handled: true,
			}
		}
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory item",
			Handled: false,
		}
	}

	// Ya tenés el item; no hace falta chequear campos vacíos
	var count int64
	if err := r.DB.Model(&database.Inventory{}).
		Where("sku = ? AND location = ? AND id <> ?", item.SKU, item.Location, inventory.ID).
		Count(&count).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check for duplicate inventory",
			Handled: false,
		}
	}

	if count > 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: fmt.Sprintf(`SKU %q already exists in location %q. Use a different location or update the existing entry.`, item.SKU, item.Location),
			Handled: true,
		}
	}

	// 2 - Update inventory
	if err := r.DB.Model(&inventory).Updates(&inventory).Where("id = ?", inventory.ID).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update inventory",
			Handled: false,
		}
	}

	// Handle lots and serials updates if necessary (similar logic to creation)
	var article database.Article
	err = r.DB.Where("sku = ?", item.SKU).First(&article).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article for inventory update",
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

	// Update lots if applicable
	if article.TrackByLot && item.DefaultLotNumber != nil {
		// Check if the lot already exists
		var lotCount int64
		err := r.DB.Model(&database.Lot{}).
			Where("lot_number = ? AND sku = ?", *item.DefaultLotNumber, item.SKU).
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
				LotNumber: *item.DefaultLotNumber,
				SKU:       item.SKU,
				Quantity:  item.Quantity,
				CreatedAt: tools.GetCurrentTime(),
				UpdatedAt: tools.GetCurrentTime(),
			}

			if err := r.DB.Create(lot).Error; err != nil {
				return &responses.InternalResponse{
					Error:   err,
					Message: "Failed to create lot",
					Handled: false,
				}
			}
		}
	}

	// Update serials if applicable
	if article.TrackBySerial && item.SerialNumberPrefix != nil {
		// Check if the serial already exists
		var serialCount int64
		err := r.DB.Model(&database.Serial{}).
			Where("serial_number LIKE ? AND sku = ?", fmt.Sprintf("%s%%", *item.SerialNumberPrefix), item.SKU).
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
				SerialNumber: *item.SerialNumberPrefix, // Assuming prefix is the full serial number for simplicity
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
	}

	return nil
}

func (s *InventoryRepository) DeleteInventory(sku, location string) *responses.InternalResponse {
	// Get the inventory item
	var inventory database.Inventory
	err := s.DB.Where("sku = ? AND location = ?", sku, location).First(&inventory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{
				Error:   nil,
				Message: "Inventory item not found",
				Handled: true,
			}
		}
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory item",
			Handled: false,
		}
	}

	// Delete serial and serial associations
	var serials []database.InventorySerial
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&serials).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory serials",
			Handled: false,
		}
	}

	for _, invSerial := range serials {
		err = s.DB.Where("serial_id = ?", invSerial.SerialID).Delete(&database.InventorySerial{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Failed to delete inventory serial association",
				Handled: false,
			}
		}
	}

	var inventorySerials []database.InventorySerial
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&inventorySerials).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory serials",
			Handled: false,
		}
	}

	for _, invSerial := range inventorySerials {
		err = s.DB.Where("serial_id = ?", invSerial.SerialID).Delete(&database.InventorySerial{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Failed to delete inventory serial association",
				Handled: false,
			}
		}
	}

	// Delete lots and lot associations
	var lots []database.InventoryLot
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&lots).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory lots",
			Handled: false,
		}
	}

	for _, invLot := range lots {
		err = s.DB.Where("lot_id = ?", invLot.LotID).Delete(&database.InventoryLot{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Failed to delete inventory lot association",
				Handled: false,
			}
		}
	}

	var inventoryLots []database.InventoryLot
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&inventoryLots).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory lots",
			Handled: false,
		}
	}

	for _, invLot := range inventoryLots {
		err = s.DB.Where("lot_id = ?", invLot.LotID).Delete(&database.InventoryLot{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Failed to delete inventory lot association",
				Handled: false,
			}
		}
	}

	// Finally, delete the inventory item itself
	if err := s.DB.Delete(&inventory).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to delete inventory item",
			Handled: false,
		}
	}

	return nil
}
