package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
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
