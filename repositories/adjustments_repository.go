package repositories

import (
	"bytes"
	"errors"
	"fmt"
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

// GetAllAdjustments returns all adjustments without tenant filter.
// internal use only — bypass tenant. Prefer GetAllForTenant in HTTP handlers.
func (r *AdjustmentsRepository) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	var adjustments []database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Order("created_at ASC").
		Find(&adjustments).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los ajustes",
			Handled: false,
		}
	}

	return adjustments, nil
}

// GetAllForTenant returns adjustments scoped to a specific tenant (S2.5 M3.1).
func (r *AdjustmentsRepository) GetAllForTenant(tenantID string) ([]database.Adjustment, *responses.InternalResponse) {
	var adjustments []database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&adjustments).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los ajustes",
			Handled: false,
		}
	}

	return adjustments, nil
}

func (r *AdjustmentsRepository) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Ajuste no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el ajuste",
			Handled: false,
		}
	}

	return &adjustment, nil
}

func (r *AdjustmentsRepository) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Message:    "Ajuste no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los detalles del ajuste",
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
				Message:    "Inventario no encontrado para este ajuste",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los detalles del inventario",
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
			Message: "Error al obtener los lotes para el inventario",
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
			Message: "Error al obtener los seriales para el inventario",
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
				Message:    "Artículo no encontrado para este ajuste",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los detalles del artículo",
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

func (r *AdjustmentsRepository) CreateAdjustment(userId string, tenantID string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	var created *database.Adjustment
	err := r.DB.Transaction(func(tx *gorm.DB) error {

		// Get inventory
		var inventory database.Inventory

		err := tx.
			Table(database.Inventory{}.TableName()).
			Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
			First(&inventory).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New("inventario no encontrado para este ajuste")
			}

			return errors.New("error al obtener los detalles del inventario")
		}

		adjustmentQuantity := adjustment.AdjustmentQuantity
		currentQuantity := inventory.Quantity
		newQuantity := currentQuantity + adjustmentQuantity

		// count_reconcile is allowed to set any qty (including below reserved or below 0 — physical reality).
		// decrease/increase checks are enforced by the service before this point.
		isCountReconcile := adjustment.AdjustmentType == "count_reconcile"
		if !isCountReconcile {
			if newQuantity < 0 {
				return errors.New("la cantidad de ajuste resulta en un inventario negativo")
			}
			// B3e (A6): block adjustment if new qty would fall below reserved_qty.
			if newQuantity < inventory.ReservedQty {
				return fmt.Errorf(
					"no puede ajustar a %.2f — hay %.2f uds reservadas en pickings activos. Cancele los pickings antes de ajustar",
					newQuantity, inventory.ReservedQty,
				)
			}
		}

		adjType := adjustment.AdjustmentType
		if adjType == "" {
			adjType = "increase"
		}

		// Create the adjustment record
		adjID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate adjustment id: %w", err)
		}
		newAdjustment := database.Adjustment{
			ID:               adjID,
			SKU:              adjustment.SKU,
			Location:         adjustment.Location,
			PreviousQuantity: int(math.Round(float64(currentQuantity))),
			AdjustmentQty:    int(math.Round(float64(adjustmentQuantity))),
			NewQuantity:      int(math.Round(float64(newQuantity))),
			Reason:           adjustment.Reason,
			Notes:            &adjustment.Notes,
			UserID:           userId,
			AdjustmentType:   adjType,
			TenantID:         tenantID, // S2.5 M3.1
		}

		err = tx.
			Table(newAdjustment.TableName()).
			Create(&newAdjustment).Error

		if err != nil {
			return errors.New("error al crear el ajuste")
		}
		created = &newAdjustment

		// Update inventory
		inventory.Quantity = newQuantity
		err = tx.
			Table(inventory.TableName()).
			Save(&inventory).Error

		if err != nil {
			return errors.New("error al actualizar el inventario")
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
					return errors.New("artículo no encontrado para este ajuste")
				}
				return errors.New("error al obtener los detalles del artículo")
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
						return errors.New("error al obtener los detalles del lote")
					}

					// If lot does not exist, create it
					if err == gorm.ErrRecordNotFound {
						adjLotID, lotIDErr := tools.GenerateNanoid(tx)
						if lotIDErr != nil {
							return fmt.Errorf("generate lot id: %w", lotIDErr)
						}
						lot = database.Lot{
							ID:             adjLotID,
							LotNumber:      adjustment.Lots[i].LotNumber,
							SKU:            adjustment.SKU,
							Quantity:       lotQuantity,
							ExpirationDate: adjustment.Lots[i].ExpirationDate,
						}

						err = tx.Table(lot.TableName()).Create(&lot).Error
						if err != nil {
							return errors.New("error al crear el lote")
						}

						// Create associate the lot with the adjustment
						adjInvLotID, adjILErr := tools.GenerateNanoid(tx)
						if adjILErr != nil {
							return fmt.Errorf("generate inventory_lot id: %w", adjILErr)
						}
						inventoryLot := database.InventoryLot{
							ID:          adjInvLotID,
							InventoryID: inventory.ID,
							LotID:       lot.ID,
							Quantity:    lotQuantity,
							Location:    adjustment.Location,
						}

						err = tx.Table(inventoryLot.TableName()).Create(&inventoryLot).Error
						if err != nil {
							return errors.New("error al asociar el lote con el inventario")
						}
					} else {
						// Update existing lot
						lot.Quantity += lotQuantity
						err = tx.Table(lot.TableName()).Save(&lot).Error
						if err != nil {
							return errors.New("error al actualizar el lote")
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
						return errors.New("error al crear la serie")
					}

					// Associate the serial with the inventory
					inventorySerial := database.InventorySerial{
						InventoryID: inventory.ID,
						SerialID:    newSerial.ID,
						Location:    adjustment.Location,
					}

					err = tx.Table(inventorySerial.TableName()).Create(&inventorySerial).Error
					if err != nil {
						return errors.New("error al asociar la serie con el inventario")
					}
				}
			}
		}

		// Create inventory movement (M3 retrofit: reference_type/id, before/after, user_id)
		adjMovID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate adjustment movement id: %w", err)
		}
		refType := "adjustment"
		refID := created.ID
		beforeQtyAdj := currentQuantity
		afterQtyAdj := newQuantity
		movements := database.InventoryMovement{
			ID:             adjMovID,
			SKU:            adjustment.SKU,
			Location:       adjustment.Location,
			MovementType:   "adjustment",
			Quantity:       adjustment.AdjustmentQuantity,
			RemainingStock: newQuantity,
			Reason:         &adjustment.Reason,
			CreatedBy:      userId,
			CreatedAt:      tools.GetCurrentTime(),
			ReferenceType:  &refType,
			ReferenceID:    &refID,
			BeforeQty:      &beforeQtyAdj,
			AfterQty:       &afterQtyAdj,
			UserID:         &userId,
		}

		err = tx.Table(database.InventoryMovement{}.TableName()).Create(&movements).Error
		if err != nil {
			return errors.New("error al crear el movimiento de inventario")
		}

		return nil
	})

	if err != nil {
		handledErrors := map[string]bool{
			"inventario no encontrado para este ajuste": true,
			"artículo no encontrado para este ajuste":   true,
		}

		errorMessage := err.Error()
		isHandled := handledErrors[errorMessage]

		if strings.Contains(errorMessage, "duplicate key value") {
			isHandled = true
			errorMessage = "El registro ya existe en la base de datos"
		}

		statusCode := 0
		if isHandled {
			if strings.Contains(errorMessage, "no encontrado") {
				statusCode = responses.StatusNotFound
			} else if strings.Contains(errorMessage, "duplicate") || strings.Contains(errorMessage, "ya existe") {
				statusCode = responses.StatusConflict
			}
		}
		return nil, &responses.InternalResponse{
			Error:      err,
			Message:    errorMessage,
			Handled:    isHandled,
			StatusCode: statusCode,
		}
	}

	return created, nil
}

func (r *AdjustmentsRepository) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	adjustments, errResp := r.GetAllAdjustments()
	if errResp != nil {
		return nil, errResp
	}

	if len(adjustments) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No hay ajustes para exportar",
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
			Message: "Error al generar el archivo Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

// GetInventoryForAdjustment returns the inventory record for a SKU+location pair (Track A stub).
func (r *AdjustmentsRepository) GetInventoryForAdjustment(sku, location string) (*database.Inventory, *responses.InternalResponse) {
	var inv database.Inventory
	if err := r.DB.Where("sku = ? AND location = ?", sku, location).First(&inv).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Inventario no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return &inv, nil
}
