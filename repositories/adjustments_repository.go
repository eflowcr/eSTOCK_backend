package repositories

import (
	"math"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type AdjustmentsRepository struct {
	DB *gorm.DB
}

func (r *AdjustmentsRepository) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	var adjustments []database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Order("created_at ASC").
		Find(&adjustments).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustments",
			Handled: false,
		}
	}

	if len(adjustments) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No adjustments found",
			Handled: true,
		}
	}

	return adjustments, nil
}

func (r *AdjustmentsRepository) GetAdjustmentByID(id int) (*database.Adjustment, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Adjustment not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustment",
			Handled: false,
		}
	}

	return &adjustment, nil
}

func (r *AdjustmentsRepository) GetAdjustmentDetails(id int) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Adjustment not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustment details",
			Handled: false,
		}
	}

	// Get inventory
	var inventory database.Inventory

	err = r.DB.
		Table(database.Inventory{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		First(&inventory).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Inventory not found for this adjustment",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory details",
			Handled: false,
		}
	}

	// Get lots for inventory
	var lots []database.Lot

	err = r.DB.
		Table(database.Lot{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		Find(&lots).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots for inventory",
			Handled: false,
		}
	}

	// Get serials for inventory
	var serials []database.Serial

	err = r.DB.
		Table(database.Serial{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		Find(&serials).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch serials for inventory",
			Handled: false,
		}
	}

	// Get article
	var article database.Article
	err = r.DB.
		Table(database.Article{}.TableName()).
		Where("sku = ?", adjustment.SKU).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Article not found for this adjustment",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article details",
			Handled: false,
		}
	}

	details := dto.AdjustmentDetails{
		Adjustment: adjustment,
		Inventory:  inventory,
		Lots:       lots,
		Serials:    serials,
		Article:    article,
	}

	return &details, nil
}

func (r *AdjustmentsRepository) CreateAdjustment(userId string, adjustment requests.CreateAdjustment) *responses.InternalResponse {
	// Get inventory
	var inventory database.Inventory

	err := r.DB.
		Table(database.Inventory{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		First(&inventory).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &responses.InternalResponse{
				Error:   nil,
				Message: "Inventory not found for this adjustment",
				Handled: true,
			}
		}
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory details",
			Handled: false,
		}
	}

	adjustmentQuantity := adjustment.AdjustmentQuantity
	currentQuantity := inventory.Quantity
	newQuantity := currentQuantity + adjustmentQuantity

	if newQuantity < 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Adjustment quantity results in negative inventory",
			Handled: true,
		}
	}

	// Create the adjustment record
	newAdjustment := database.Adjustment{
		SKU:              adjustment.SKU,
		Location:         adjustment.Location,
		PreviousQuantity: int(math.Round(float64(currentQuantity))),
		AdjustmentQty:    int(math.Round(float64(adjustmentQuantity))),
		NewQuantity:      int(math.Round(float64(newQuantity))),
		Reason:           adjustment.Reason,
		Notes:            &adjustment.Notes,
		UserID:           userId,
	}

	err = r.DB.
		Table(newAdjustment.TableName()).
		Create(&newAdjustment).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create adjustment",
			Handled: false,
		}
	}

	// Update inventory
	inventory.Quantity = newQuantity
	err = r.DB.
		Table(inventory.TableName()).
		Save(&inventory).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update inventory",
			Handled: false,
		}
	}

	// Handle lots and serials
	if adjustmentQuantity > 0 {
		// Get article by SKU
		var article database.Article

		err = r.DB.
			Table(database.Article{}.TableName()).
			Where("sku = ?", adjustment.SKU).
			First(&article).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return &responses.InternalResponse{
					Error:   nil,
					Message: "Article not found for this adjustment",
					Handled: true,
				}
			}
			return &responses.InternalResponse{
				Error:   err,
				Message: "Failed to fetch article details",
				Handled: false,
			}
		}

		if article.TrackByLot && adjustment.Lots != nil {
			for i := 0; i < len(adjustment.Lots); i++ {
				lotQuantity := float64(adjustment.Lots[i].Quantity)

				// Get existing lot
				var lot database.Lot
				err = r.DB.
					Table(database.Lot{}.TableName()).
					Where("sku = ? AND location = ? AND lot_number = ?", adjustment.SKU, adjustment.Location, adjustment.Lots[i].LotNumber).
					First(&lot).Error

				if err != nil && err != gorm.ErrRecordNotFound {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to fetch lot details",
						Handled: false,
					}
				}

				// If lot does not exist, create it
				if err == gorm.ErrRecordNotFound {
					lot = database.Lot{
						LotNumber:      adjustment.Lots[i].LotNumber,
						SKU:            adjustment.SKU,
						Quantity:       lotQuantity,
						ExpirationDate: adjustment.Lots[i].ExpirationDate,
					}

					err = r.DB.Table(lot.TableName()).Create(&lot).Error
					if err != nil {
						return &responses.InternalResponse{
							Error:   err,
							Message: "Failed to create lot",
							Handled: false,
						}
					}

					// Create associate the lot with the adjustment
					inventoryLot := database.InventoryLot{
						InventoryID: inventory.ID,
						LotID:       lot.ID,
						Quantity:    lotQuantity,
						Location:    adjustment.Location,
					}

					err = r.DB.Table(inventoryLot.TableName()).Create(&inventoryLot).Error
					if err != nil {
						return &responses.InternalResponse{
							Error:   err,
							Message: "Failed to associate lot with inventory",
							Handled: false,
						}
					}
				} else {
					// Update existing lot
					lot.Quantity += lotQuantity
					err = r.DB.Table(lot.TableName()).Save(&lot).Error
					if err != nil {
						return &responses.InternalResponse{
							Error:   err,
							Message: "Failed to update lot",
							Handled: false,
						}
					}
				}
			}
		}

		if article.TrackBySerial && adjustment.Serials != nil {
			for i := 0; i < len(adjustment.Serials); i++ {
				newSerial := database.Serial{
					SerialNumber: adjustment.Serials[i],
					SKU:          adjustment.SKU,
					Status:       "available",
				}

				err = r.DB.Table(newSerial.TableName()).Create(&newSerial).Error
				if err != nil {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to create serial",
						Handled: false,
					}
				}

				// Associate the serial with the inventory
				inventorySerial := database.InventorySerial{
					InventoryID: inventory.ID,
					SerialID:    newSerial.ID,
					Location:    adjustment.Location,
				}

				err = r.DB.Table(inventorySerial.TableName()).Create(&inventorySerial).Error
				if err != nil {
					return &responses.InternalResponse{
						Error:   err,
						Message: "Failed to associate serial with inventory",
						Handled: false,
					}
				}
			}
		}
	}

	return nil
}
