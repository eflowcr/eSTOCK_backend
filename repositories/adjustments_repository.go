package repositories

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
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
	err := r.DB.Transaction(func(tx *gorm.DB) error {

		// Get inventory
		var inventory database.Inventory

		err := tx.
			Table(database.Inventory{}.TableName()).
			Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
			First(&inventory).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New("Inventory not found for this adjustment")
			}

			return errors.New("Failed to fetch inventory details")
		}

		adjustmentQuantity := adjustment.AdjustmentQuantity
		currentQuantity := inventory.Quantity
		newQuantity := currentQuantity + adjustmentQuantity

		if newQuantity < 0 {
			return errors.New("Adjustment quantity results in negative inventory")
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

		err = tx.
			Table(newAdjustment.TableName()).
			Create(&newAdjustment).Error

		if err != nil {
			return errors.New("Failed to create adjustment")
		}

		// Update inventory
		inventory.Quantity = newQuantity
		err = tx.
			Table(inventory.TableName()).
			Save(&inventory).Error

		if err != nil {
			return errors.New("Failed to update inventory")
		}

		// Handle lots and serials
		if adjustmentQuantity > 0 {
			// Get article by SKU
			var article database.Article

			err = tx.
				Table(database.Article{}.TableName()).
				Where("sku = ?", adjustment.SKU).
				First(&article).Error

			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return errors.New("Article not found for this adjustment")
				}
				return errors.New("Failed to fetch article details")
			}

			if article.TrackByLot && adjustment.Lots != nil {
				for i := 0; i < len(adjustment.Lots); i++ {
					lotQuantity := float64(adjustment.Lots[i].Quantity)

					// Get existing lot
					var lot database.Lot
					err = tx.
						Table(database.Lot{}.TableName()).
						Where("sku = ? AND lot_number = ?", adjustment.SKU, adjustment.Lots[i].LotNumber).
						First(&lot).Error

					if err != nil && err != gorm.ErrRecordNotFound {
						return errors.New("Failed to fetch lot details")
					}

					// If lot does not exist, create it
					if err == gorm.ErrRecordNotFound {
						lot = database.Lot{
							LotNumber:      adjustment.Lots[i].LotNumber,
							SKU:            adjustment.SKU,
							Quantity:       lotQuantity,
							ExpirationDate: adjustment.Lots[i].ExpirationDate,
						}

						err = tx.Table(lot.TableName()).Create(&lot).Error
						if err != nil {
							return errors.New("Failed to create lot")
						}

						// Create associate the lot with the adjustment
						inventoryLot := database.InventoryLot{
							InventoryID: inventory.ID,
							LotID:       lot.ID,
							Quantity:    lotQuantity,
							Location:    adjustment.Location,
						}

						err = tx.Table(inventoryLot.TableName()).Create(&inventoryLot).Error
						if err != nil {
							return errors.New("Failed to associate lot with inventory")
						}
					} else {
						// Update existing lot
						lot.Quantity += lotQuantity
						err = tx.Table(lot.TableName()).Save(&lot).Error
						if err != nil {
							return errors.New("Failed to update lot")
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

					err = tx.Table(newSerial.TableName()).Create(&newSerial).Error
					if err != nil {
						return errors.New("Failed to create serial")
					}

					// Associate the serial with the inventory
					inventorySerial := database.InventorySerial{
						InventoryID: inventory.ID,
						SerialID:    newSerial.ID,
						Location:    adjustment.Location,
					}

					err = tx.Table(inventorySerial.TableName()).Create(&inventorySerial).Error
					if err != nil {
						return errors.New("Failed to associate serial with inventory")
					}
				}
			}
		}

		// Create inventory movement
		movements := database.InventoryMovement{
			SKU:            adjustment.SKU,
			Location:       adjustment.Location,
			MovementType:   "adjustment",
			Quantity:       adjustment.AdjustmentQuantity,
			RemainingStock: newQuantity,
			Reason:         &adjustment.Reason,
			CreatedBy:      userId,
			CreatedAt:      tools.GetCurrentTime(),
		}

		err = tx.Table(database.InventoryMovement{}.TableName()).Create(&movements).Error
		if err != nil {
			return errors.New("Failed to create inventory movement")
		}

		return nil
	})

	if err != nil {
		handledErrors := map[string]bool{
			"Inventory not found for this adjustment": true,
			"Article not found for this adjustment":   true,
		}

		errorMessage := err.Error()
		isHandled := handledErrors[errorMessage]

		if strings.Contains(errorMessage, "duplicate key value") {
			isHandled = true
			errorMessage = "El registro ya existe en la base de datos"
		}

		return &responses.InternalResponse{
			Error:   err,
			Message: errorMessage,
			Handled: isHandled,
		}
	}

	return nil
}

func (r *AdjustmentsRepository) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	adjustments, errResp := r.GetAllAdjustments()
	if errResp != nil {
		return nil, errResp
	}

	if len(adjustments) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No adjustments found to export",
			Handled: true,
		}
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID",
		"SKU",
		"Ubicación",
		"Cantidad Anterior",
		"Ajuste",
		"Nueva Cantidad",
		"Motivo",
		"Notas",
		"Usuario",
		"Creado En",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		_ = f.SetCellValue(sheet, cell, h)
	}

	strOrEmpty := func(p *string) string {
		if p != nil {
			return *p
		}
		return ""
	}

	for idx, a := range adjustments {
		row := idx + 7
		values := []interface{}{
			a.ID,
			a.SKU,
			a.Location,
			a.PreviousQuantity,
			a.AdjustmentQty,
			a.NewQuantity,
			a.Reason,
			strOrEmpty(a.Notes),
			a.UserID,
			a.CreatedAt.Format(time.RFC3339),
		}

		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate Excel file",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}
