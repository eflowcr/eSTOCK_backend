package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
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

type InventoryRepository struct {
	DB *gorm.DB
}

func (r *InventoryRepository) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	var items []database.Inventory
	err := r.DB.Where("quantity > 0").
		Order("sku ASC").
		Find(&items).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener inventario",
			Handled: false,
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
				Message: "Error al obtener artículo para el elemento de inventario",
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
					Message: "Error al obtener lotes para el elemento de inventario",
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
					Message: "Error al obtener seriales para el elemento de inventario",
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
			UnitPrice:       item.UnitPrice,
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

func (r *InventoryRepository) CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse {
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// 1 - Check if sku exists in the location
		var inventoryCount int64
		err := r.DB.Model(&database.Inventory{}).
			Where("sku = ? AND location = ?", item.SKU, item.Location).
			Count(&inventoryCount).Error

		if err != nil {
			return errors.New("error al verificar inventario existente")
		}

		if inventoryCount > 0 {
			return errors.New("el inventario con este SKU ya existe en la ubicación especificada")
		}

		// 2 - Get article information
		var article database.Article
		err = r.DB.Where("sku = ?", item.SKU).First(&article).Error
		if err != nil {
			return errors.New("error al obtener artículo para la creación de inventario")
		}

		if article.ID == 0 {
			return errors.New("artículo no encontrado para el SKU proporcionado")
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
			return errors.New("error al crear inventario")
		}

		// 3 - Create lots if applicable
		if article.TrackByLot && item.Lots != nil {
			for i := 0; i < len(item.Lots); i++ {
				var lotCount int64

				err := r.DB.Model(&database.Lot{}).
					Where("lot_number = ? AND sku = ?", item.Lots[i].LotNumber, item.SKU).
					Count(&lotCount).Error

				if err != nil {
					return errors.New("error al verificar lote existente")
				}

				if lotCount == 0 {
					// String to time.Time
					var expirationDate time.Time

					if item.Lots[i].ExpirationDate != nil {
						expirationDate, _ = time.Parse("2006-01-02", *item.Lots[i].ExpirationDate)
					}

					// Create new lot
					lot := &database.Lot{
						LotNumber:      item.Lots[i].LotNumber,
						SKU:            item.SKU,
						Quantity:       item.Lots[i].Quantity,
						ExpirationDate: &expirationDate,
						CreatedAt:      tools.GetCurrentTime(),
						UpdatedAt:      tools.GetCurrentTime(),
					}

					if err := r.DB.Create(lot).Error; err != nil {
						return errors.New("error al crear lote")
					}

					// Create inventory_lot association
					inventoryLot := &database.InventoryLot{
						InventoryID: inventory.ID,
						LotID:       lot.ID,
						Quantity:    item.Lots[i].Quantity,
						Location:    item.Location,
					}

					if err := r.DB.Create(inventoryLot).Error; err != nil {
						return errors.New("error al crear asociación de inventario_lote")
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
					return errors.New("error al verificar serial existente")
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
						return errors.New("error al crear serial")
					}

					// Create inventory_serial association
					inventorySerial := &database.InventorySerial{
						InventoryID: inventory.ID,
						SerialID:    newSerial.ID,
						Location:    item.Location,
					}

					if err := r.DB.Create(inventorySerial).Error; err != nil {
						return errors.New("error al crear asociación de inventario_serial")
					}
				}
			}
		}

		// Reason
		reason := "in"

		// 5 - Create inventory movement
		inventoryMovement := &database.InventoryMovement{
			SKU:            item.SKU,
			Location:       item.Location,
			MovementType:   reason,
			Quantity:       item.Quantity,
			RemainingStock: item.Quantity,
			Reason:         &reason,
			CreatedBy:      userId,
			CreatedAt:      tools.GetCurrentTime(),
		}

		if err := r.DB.Create(inventoryMovement).Error; err != nil {
			return errors.New("error al crear movimiento de inventario")
		}

		return nil
	})

	if err != nil {
		handledErrors := map[string]bool{
			"el inventario con este SKU ya existe en la ubicación especificada": true,
			"artículo no encontrado para el SKU proporcionado":                  true,
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

func (r *InventoryRepository) UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse {
	// 1 - Get the current inventory item
	var inventory database.Inventory
	err := r.DB.Where("sku = ? AND location = ?", item.SKU, item.Location).First(&inventory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{
				Error:   nil,
				Message: "Artículo de inventario no encontrado",
				Handled: true,
			}
		}
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener artículo de inventario",
			Handled: false,
		}
	}

	var count int64
	if err := r.DB.Model(&database.Inventory{}).
		Where("sku = ? AND location = ? AND id <> ?", item.SKU, item.Location, inventory.ID).
		Count(&count).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al verificar inventario duplicado",
			Handled: false,
		}
	}

	if count > 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: fmt.Sprintf(`SKU %q ya existe en la ubicación %q. Use una ubicación diferente o actualice la entrada existente.`, item.SKU, item.Location),
			Handled: true,
		}
	}

	// 2 - Update inventory
	inventory.Name = item.Name
	inventory.Description = item.Description
	inventory.Location = item.Location
	inventory.Quantity = item.Quantity
	inventory.UnitPrice = item.UnitPrice
	inventory.Status = item.Status
	inventory.UpdatedAt = tools.GetCurrentTime()

	if err := r.DB.Model(&inventory).Updates(&inventory).Where("id = ?", inventory.ID).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar inventario",
			Handled: false,
		}
	}

	// Handle lots and serials updates if necessary (similar logic to creation)
	var article database.Article
	err = r.DB.Where("sku = ?", item.SKU).First(&article).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener artículo para actualización de inventario",
			Handled: false,
		}
	}

	if article.ID == 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Artículo no encontrado para el SKU proporcionado",
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
				Message: "Error al verificar lote existente",
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
					Message: "Error al crear lote",
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
				Message: "Error al verificar número de serie existente",
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
					Message: "Error al crear número de serie",
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
				Message: "Artículo de inventario no encontrado",
				Handled: true,
			}
		}
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener artículo de inventario",
			Handled: false,
		}
	}

	// Delete serial and serial associations
	var serials []database.InventorySerial
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&serials).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener números de serie de inventario",
			Handled: false,
		}
	}

	for _, invSerial := range serials {
		err = s.DB.Where("serial_id = ?", invSerial.SerialID).Delete(&database.InventorySerial{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Error al eliminar asociación de número de serie de inventario",
				Handled: false,
			}
		}
	}

	var inventorySerials []database.InventorySerial
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&inventorySerials).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener números de serie de inventario",
			Handled: false,
		}
	}

	for _, invSerial := range inventorySerials {
		err = s.DB.Where("serial_id = ?", invSerial.SerialID).Delete(&database.InventorySerial{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Error al eliminar asociación de número de serie de inventario",
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
			Message: "Error al obtener lotes de inventario",
			Handled: false,
		}
	}

	for _, invLot := range lots {
		err = s.DB.Where("lot_id = ?", invLot.LotID).Delete(&database.InventoryLot{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Error al eliminar asociación de lote de inventario",
				Handled: false,
			}
		}
	}

	var inventoryLots []database.InventoryLot
	err = s.DB.Where("inventory_id = ?", inventory.ID).Find(&inventoryLots).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener lotes de inventario",
			Handled: false,
		}
	}

	for _, invLot := range inventoryLots {
		err = s.DB.Where("lot_id = ?", invLot.LotID).Delete(&database.InventoryLot{}).Error
		if err != nil {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Error al eliminar asociación de lote de inventario",
				Handled: false,
			}
		}
	}

	// Finally, delete the inventory item itself
	if err := s.DB.Delete(&inventory).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar artículo de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) Trend(sku string) (*dto.ConsumptionTrend, *responses.InternalResponse) {
	days := 30

	if days <= 0 {
		days = 30
	}

	now := time.Now().UTC()
	cutoffDate := now.AddDate(0, 0, -days)

	var movements []database.InventoryMovement
	if err := r.DB.
		Where("sku = ? AND movement_type = ? AND created_at >= ?", sku, "outbound", cutoffDate).
		Order("created_at ASC").
		Find(&movements).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener movimientos de inventario",
			Handled: false,
		}
	}

	if len(movements) == 0 {
		return &dto.ConsumptionTrend{
			AverageDailyConsumption: 0,
			Trend:                   "stable",
			PredictedStockOutDays:   -1,
		}, nil
	}

	var totalConsumption float64
	for _, m := range movements {
		if !strings.EqualFold(strings.TrimSpace(m.MovementType), "outbound") {
			continue
		}
		totalConsumption += math.Abs(float64(m.Quantity))
	}
	avgDaily := totalConsumption / float64(days)

	mid := len(movements) / 2
	older := movements[:mid]
	recent := movements[mid:]

	var olderSum, recentSum float64
	for _, m := range older {
		olderSum += math.Abs(float64(m.Quantity))
	}
	for _, m := range recent {
		recentSum += math.Abs(float64(m.Quantity))
	}

	var olderAvg, recentAvg float64
	if len(older) > 0 {
		olderAvg = olderSum / float64(len(older))
	}
	if len(recent) > 0 {
		recentAvg = recentSum / float64(len(recent))
	}

	trendStr := "stable"
	if recentAvg > olderAvg*1.10 {
		trendStr = "increasing"
	} else if recentAvg < olderAvg*0.90 {
		trendStr = "decreasing"
	}

	currentStock := 0.0
	var inv database.Inventory
	if err := r.DB.Where("sku = ?", sku).First(&inv).Error; err == nil {
		currentStock = inv.Quantity
	}

	predicted := -1.0
	if avgDaily > 0 {
		predicted = currentStock / avgDaily
	}

	return &dto.ConsumptionTrend{
		AverageDailyConsumption: avgDaily,
		Trend:                   trendStr,
		PredictedStockOutDays:   predicted,
	}, nil
}

func (r *InventoryRepository) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []*responses.InternalResponse) {
	imported := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return imported, []*responses.InternalResponse{{
			Error:   err,
			Message: "Error al abrir archivo Excel",
			Handled: false,
		}}
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return imported, []*responses.InternalResponse{{
			Error:   err,
			Message: "Error al leer filas de la hoja de Excel",
			Handled: false,
		}}
	}

	for i, row := range rows {
		if i < 7 || len(row) < 10 {
			continue
		}

		sku := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		description := strings.TrimSpace(row[2])
		location := strings.TrimSpace(row[3])
		quantityStr := strings.TrimSpace(row[4])
		unitPriceStr := strings.TrimSpace(row[5])
		trackByLot := strings.EqualFold(strings.TrimSpace(row[6]), "Si")
		trackBySerial := strings.EqualFold(strings.TrimSpace(row[7]), "Si")

		if sku == "" || location == "" || quantityStr == "" {
			continue
		}

		quantity, _ := strconv.Atoi(quantityStr)

		var unitPrice *float64
		if unitPriceStr != "" {
			if f, err := strconv.ParseFloat(unitPriceStr, 64); err == nil {
				unitPrice = &f
			}
		}

		var descPtr *string
		if description != "" {
			descPtr = &description
		}

		// Armar la estructura de inventario
		item := &requests.CreateInventory{
			SKU:         sku,
			Name:        name,
			Description: descPtr,
			Location:    location,
			Quantity:    float64(quantity),
			UnitPrice:   unitPrice,
		}

		// Lotes
		if trackByLot {
			var lots []requests.CreateLotRequest
			for j := i; j < len(rows); j++ {
				if len(rows[j]) < 14 {
					continue
				}
				if strings.TrimSpace(rows[j][11]) != sku {
					continue
				}

				lotQty, _ := strconv.Atoi(strings.TrimSpace(rows[j][12]))
				expirationDate := strings.TrimSpace(rows[j][13])

				// Convert expiration date to time.Time
				var expDate *time.Time
				if expirationDate != "" {
					if date, err := time.Parse("2006-01-02", expirationDate); err == nil {
						expDate = &date
					}
				}

				// expDate as *string
				var expDateStr *string
				if expDate != nil {
					formatted := expDate.Format("2006-01-02")
					expDateStr = &formatted
				}

				lots = append(lots, requests.CreateLotRequest{
					LotNumber:      strings.TrimSpace(rows[j][10]),
					SKU:            sku,
					Quantity:       float64(lotQty),
					ExpirationDate: expDateStr,
				})
			}
			item.Lots = lots
		}

		// Seriales
		if trackBySerial {
			var serials []database.Serial
			for j := i; j < len(rows); j++ {
				if len(rows[j]) < 16 {
					continue
				}
				if strings.TrimSpace(rows[j][15]) != sku {
					continue
				}

				serials = append(serials, database.Serial{
					SerialNumber: strings.TrimSpace(rows[j][14]),
					SKU:          sku,
					Status:       "available",
				})
			}
			item.Serials = serials
		}

		// Crear el inventario
		resp := r.CreateInventory(userId, item)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Row %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}

		imported = append(imported, sku)
	}

	return imported, errorsList
}

func (r *InventoryRepository) ExportInventoryToExcel() ([]byte, *responses.InternalResponse) {
	inventory, errResp := r.GetAllInventory()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"SKU", "Name", "Description", "Location", "Quantity", "Unit Price",
		"Track by Lot", "Track by Serial", "Track Expiration", "Min Quantity",
		"Max Quantity", "Image URL", "Lot Number", "Lot Quantity", "Lot Expiration Date",
		"Serial Number",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, item := range inventory {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.SKU)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Name)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.Description)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.Location)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Quantity)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.UnitPrice)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), boolToSiNo(item.TrackByLot))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), boolToSiNo(item.TrackBySerial))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), boolToSiNo(item.TrackExpiration))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), item.MinQuantity)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), item.MaxQuantity)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), item.ImageURL)

		if len(item.Lots) > 0 {
			for j, lot := range item.Lots {
				lotRow := row + j
				f.SetCellValue(sheet, fmt.Sprintf("M%d", lotRow), lot.LotNumber)
				f.SetCellValue(sheet, fmt.Sprintf("N%d", lotRow), lot.Quantity)
				if lot.ExpirationDate != nil {
					f.SetCellValue(sheet, fmt.Sprintf("O%d", lotRow), lot.ExpirationDate.Format("2006-01-02"))
				} else {
					f.SetCellValue(sheet, fmt.Sprintf("O%d", lotRow), "")
				}
			}
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("M%d", row), "")
			f.SetCellValue(sheet, fmt.Sprintf("N%d", row), "")
			f.SetCellValue(sheet, fmt.Sprintf("O%d", row), "")
		}

		if len(item.Serials) > 0 {
			for j, serial := range item.Serials {
				serialRow := row + j
				f.SetCellValue(sheet, fmt.Sprintf("P%d", serialRow), serial.SerialNumber)
			}
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("P%d", row), "")
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al escribir archivo Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func (r *InventoryRepository) GetInventoryLots(inventoryID int) ([]responses.InventoryLot, *responses.InternalResponse) {
	var result []responses.InventoryLot

	err := r.DB.
		Table("inventory_lots").
		Select("inventory_lots.*, lots.id as lot_id, lots.lot_number, lots.sku, lots.quantity as lot_quantity, lots.expiration_date, lots.created_at as lot_created_at, lots.updated_at as lot_updated_at").
		Joins("INNER JOIN lots ON inventory_lots.lot_id = lots.id").
		Where("inventory_lots.inventory_id = ?", inventoryID).
		Scan(&result).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener lotes de inventario",
			Handled: false,
		}
	}

	return result, nil
}

func (r *InventoryRepository) GetInventorySerials(inventoryID int) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) {
	var result []responses.InventorySerialWithSerial

	err := r.DB.
		Table("inventory_serials").
		Select(`
			inventory_serials.id, inventory_serials.inventory_id, inventory_serials.serial_id,
			inventory_serials.location, inventory_serials.created_at,
			serials.id as serial_id, serials.serial_number, serials.sku, serials.status,
			serials.created_at as serial_created_at, serials.updated_at as serial_updated_at
		`).
		Joins("INNER JOIN serials ON inventory_serials.serial_id = serials.id").
		Where("inventory_serials.inventory_id = ?", inventoryID).
		Scan(&result).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener números de serie de inventario",
			Handled: false,
		}
	}

	return result, nil
}

func (r *InventoryRepository) CreateInventoryLot(id int, input *requests.CreateInventoryLotRequest) *responses.InternalResponse {
	inventoryLot := &database.InventoryLot{
		InventoryID: id,
		LotID:       input.LotID,
		Quantity:    input.Quantity,
		Location:    input.Location,
		CreatedAt:   time.Now(),
	}

	if err := r.DB.Create(inventoryLot).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear lote de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) DeleteInventoryLot(id int) *responses.InternalResponse {
	if err := r.DB.Where("id = ?", id).Delete(&database.InventoryLot{}).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar lote de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) CreateInventorySerial(id int, input *requests.CreateInventorySerial) *responses.InternalResponse {
	inventorySerial := &database.InventorySerial{
		InventoryID: id,
		SerialID:    input.SerialID,
		Location:    input.Location,
		CreatedAt:   time.Now(),
	}

	if err := r.DB.Create(inventorySerial).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear número de serie de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) DeleteInventorySerial(id int) *responses.InternalResponse {
	if err := r.DB.Where("id = ?", id).Delete(&database.InventorySerial{}).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar número de serie de inventario",
			Handled: false,
		}
	}

	return nil
}
