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

// GetInventoryBySkuAndLocation returns a single inventory record by SKU and location, or nil if not found.
func (r *InventoryRepository) GetInventoryBySkuAndLocation(sku, location string) (*dto.EnhancedInventory, *responses.InternalResponse) {
	var item database.Inventory
	err := r.DB.Where("sku = ? AND location = ?", sku, location).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener inventario por SKU y ubicación",
			Handled: false,
		}
	}

	var article database.Article
	err = r.DB.Where("sku = ?", item.SKU).First(&article).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener artículo para el elemento de inventario",
			Handled: false,
		}
	}

	var lots []database.Lot
	if article.TrackByLot {
		_ = r.DB.
			Table(database.Lot{}.TableName()).
			Joins("JOIN inventory_lots ON lots.id = inventory_lots.lot_id").
			Where("inventory_lots.inventory_id = ?", item.ID).
			Find(&lots).Error
	}

	var serials []database.Serial
	if article.TrackBySerial {
		_ = r.DB.
			Table(database.Serial{}.TableName()).
			Joins("JOIN inventory_serials ON serials.id = inventory_serials.serial_id").
			Where("inventory_serials.inventory_id = ?", item.ID).
			Find(&serials).Error
	}

	imageURL := ""
	if article.ImageURL != nil {
		imageURL = *article.ImageURL
	}

	desc := ""
	if article.Description != nil {
		desc = *article.Description
	}

	minQty, maxQty := 0, 0
	if article.MinQuantity != nil {
		minQty = *article.MinQuantity
	}
	if article.MaxQuantity != nil {
		maxQty = *article.MaxQuantity
	}

	return &dto.EnhancedInventory{
		ID:              item.ID,
		SKU:             item.SKU,
		Location:        item.Location,
		Quantity:        item.Quantity,
		Status:          item.Status,
		UnitPrice:       item.UnitPrice,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		Name:            article.Name,
		Description:     desc,
		Presentation:    article.Presentation,
		TrackByLot:      article.TrackByLot,
		TrackBySerial:   article.TrackBySerial,
		TrackExpiration: article.TrackExpiration,
		ImageURL:        imageURL,
		MinQuantity:     minQty,
		MaxQuantity:     maxQty,
		Lots:            lots,
		Serials:         serials,
	}, nil
}

func (r *InventoryRepository) CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse {
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// 1 - Check if sku exists in the location
		var inventoryCount int64
		err := tx.Model(&database.Inventory{}).
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
		err = tx.Where("sku = ?", item.SKU).First(&article).Error
		if err != nil {
			return errors.New("error al obtener artículo para la creación de inventario")
		}

		if article.ID == "" {
			return errors.New("artículo no encontrado para el SKU proporcionado")
		}

		inventoryID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generar id inventario: %w", err)
		}

		var inventory database.Inventory
		inventory.ID = inventoryID
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

		if err := tx.Create(&inventory).Error; err != nil {
			return errors.New("error al crear inventario")
		}

		// 3 - Create lots if applicable
		if article.TrackByLot && item.Lots != nil {
			for i := 0; i < len(item.Lots); i++ {
				var lotCount int64

				err := tx.Model(&database.Lot{}).
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
					lotID, err := tools.GenerateNanoid(tx)
					if err != nil {
						return fmt.Errorf("generar id lote: %w", err)
					}
					lot := &database.Lot{
						ID:             lotID,
						LotNumber:      item.Lots[i].LotNumber,
						SKU:            item.SKU,
						Quantity:       item.Lots[i].Quantity,
						ExpirationDate: &expirationDate,
						CreatedAt:      tools.GetCurrentTime(),
						UpdatedAt:      tools.GetCurrentTime(),
					}

					if err := tx.Create(lot).Error; err != nil {
						return errors.New("error al crear lote")
					}

					// Create inventory_lot association
					invLotID, err := tools.GenerateNanoid(tx)
					if err != nil {
						return fmt.Errorf("generar id inventory_lot: %w", err)
					}
					inventoryLot := &database.InventoryLot{
						ID:          invLotID,
						InventoryID: inventory.ID,
						LotID:       lot.ID,
						Quantity:    item.Lots[i].Quantity,
						Location:    item.Location,
					}

					if err := tx.Create(inventoryLot).Error; err != nil {
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
				err := tx.Model(&database.Serial{}).
					Where("serial_number = ? AND sku = ?", item.Serials[i].SerialNumber, item.SKU).
					Count(&serialCount).Error

				if err != nil {
					return errors.New("error al verificar serial existente")
				}

				if serialCount == 0 {
					// Create new serial
					serialID, err := tools.GenerateNanoid(tx)
					if err != nil {
						return fmt.Errorf("generar id serial: %w", err)
					}
					newSerial := &database.Serial{
						ID:           serialID,
						SerialNumber: item.Serials[i].SerialNumber,
						SKU:          item.SKU,
						CreatedAt:    tools.GetCurrentTime(),
						UpdatedAt:    tools.GetCurrentTime(),
						Status:       "available",
					}

					if err := tx.Create(newSerial).Error; err != nil {
						return errors.New("error al crear serial")
					}

					// Create inventory_serial association
					invSerialID, err := tools.GenerateNanoid(tx)
					if err != nil {
						return fmt.Errorf("generar id inventory_serial: %w", err)
					}
					inventorySerial := &database.InventorySerial{
						ID:          invSerialID,
						InventoryID: inventory.ID,
						SerialID:    newSerial.ID,
						Location:    item.Location,
					}

					if err := tx.Create(inventorySerial).Error; err != nil {
						return errors.New("error al crear asociación de inventario_serial")
					}
				}
			}
		}

		// Reason
		reason := "in"

		// 5 - Create inventory movement
		movementID, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generar id movimiento: %w", err)
		}
		inventoryMovement := &database.InventoryMovement{
			ID:             movementID,
			SKU:            item.SKU,
			Location:       item.Location,
			MovementType:   reason,
			Quantity:       item.Quantity,
			RemainingStock: item.Quantity,
			Reason:         &reason,
			CreatedBy:      userId,
			CreatedAt:      tools.GetCurrentTime(),
		}

		if err := tx.Create(inventoryMovement).Error; err != nil {
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

		statusCode := 0
		if isHandled {
			if strings.Contains(errorMessage, "no encontrado") {
				statusCode = responses.StatusNotFound
			} else if strings.Contains(errorMessage, "ya existe") || strings.Contains(errorMessage, "duplicate") {
				statusCode = responses.StatusConflict
			}
		}
		return &responses.InternalResponse{
			Error:      err,
			Message:    errorMessage,
			Handled:    isHandled,
			StatusCode: statusCode,
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
				Message:    "Artículo de inventario no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
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
			Message:    fmt.Sprintf(`SKU %q ya existe en la ubicación %q. Use una ubicación diferente o actualice la entrada existente.`, item.SKU, item.Location),
			Handled:    true,
			StatusCode: responses.StatusConflict,
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

	if article.ID == "" {
		return &responses.InternalResponse{
			Message:    "Artículo no encontrado para el SKU proporcionado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
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
				Message:    "Artículo de inventario no encontrado",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
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

	// Finally, delete the inventory item itself (explicit WHERE to avoid GORM batch-delete safety when primary key is empty)
	if err := s.DB.Where("sku = ? AND location = ?", sku, location).Delete(&database.Inventory{}).Error; err != nil {
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

func (r *InventoryRepository) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return imported, skipped, &responses.InternalResponse{Error: err, Message: "Error al abrir archivo Excel", Handled: false}
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return imported, skipped, &responses.InternalResponse{Message: "Sin hojas de datos", Handled: true}
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return imported, skipped, &responses.InternalResponse{Error: err, Message: "Error al leer filas", Handled: false}
	}

	for i, row := range rows {
		if i < 8 || len(row) < 10 { // Skip rows 1-8 (header/instructions/example)
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

		// Skip example row
		if strings.EqualFold(sku, "SKU-0001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
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
			return imported, skipped, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			}
		}

		imported = append(imported, sku)
	}

	return imported, skipped, nil
}

func (r *InventoryRepository) ImportInventoryFromJSON(userId string, rows []requests.InventoryImportRow) ([]string, []string, *responses.InternalResponse) {
	imported := []string{}
	skipped := []string{}

	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		location := strings.TrimSpace(row.Location)
		quantityStr := strings.TrimSpace(row.Quantity)

		if sku == "" || location == "" || quantityStr == "" {
			skipped = append(skipped, fmt.Sprintf("Fila %d: SKU, ubicación y cantidad son requeridos", i+1))
			continue
		}
		if strings.EqualFold(sku, "SKU-0001") {
			skipped = append(skipped, fmt.Sprintf("Fila %d: fila de ejemplo omitida", i+1))
			continue
		}

		qty, err := strconv.ParseFloat(quantityStr, 64)
		if err != nil || qty < 0 {
			skipped = append(skipped, fmt.Sprintf("Fila %d: cantidad inválida", i+1))
			continue
		}

		var unitPrice *float64
		if row.UnitPrice != "" {
			if p, e := strconv.ParseFloat(strings.TrimSpace(row.UnitPrice), 64); e == nil {
				unitPrice = &p
			}
		}

		desc := strings.TrimSpace(row.Description)
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}

		item := &requests.CreateInventory{
			SKU:         sku,
			Name:        strings.TrimSpace(row.Name),
			Description: descPtr,
			Location:    location,
			Quantity:    qty,
			UnitPrice:   unitPrice,
		}

		resp := r.CreateInventory(userId, item)
		if resp != nil {
			return imported, skipped, &responses.InternalResponse{
				Error: resp.Error, Message: fmt.Sprintf("Fila %d: %s", i+1, resp.Message), Handled: resp.Handled,
			}
		}
		imported = append(imported, sku)
	}
	return imported, skipped, nil
}

func (r *InventoryRepository) ValidateImportRows(rows []requests.InventoryImportRow) ([]responses.InventoryValidationResult, *responses.InternalResponse) {
	results := make([]responses.InventoryValidationResult, 0, len(rows))
	seenKeys := make(map[string]bool)

	for i, row := range rows {
		sku := strings.TrimSpace(row.SKU)
		location := strings.TrimSpace(row.Location)
		qty := strings.TrimSpace(row.Quantity)
		result := responses.InventoryValidationResult{RowIndex: i, Row: row}

		// Field validation
		if sku == "" || location == "" || qty == "" {
			result.Status = responses.InventoryStatusError
			result.FieldErrors = map[string]string{}
			if sku == "" {
				result.FieldErrors["sku"] = "SKU requerido"
			}
			if location == "" {
				result.FieldErrors["location"] = "Ubicación requerida"
			}
			if qty == "" {
				result.FieldErrors["quantity"] = "Cantidad requerida"
			}
			results = append(results, result)
			continue
		}
		if _, err := strconv.ParseFloat(qty, 64); err != nil {
			result.Status = responses.InventoryStatusError
			result.FieldErrors = map[string]string{"quantity": "Cantidad debe ser un número"}
			results = append(results, result)
			continue
		}

		// Duplicate within batch
		key := strings.ToLower(sku + "|" + location)
		if seenKeys[key] {
			result.Status = responses.InventoryStatusDuplicate
			results = append(results, result)
			continue
		}
		seenKeys[key] = true

		// Exact SKU+location match in DB
		existing, _ := r.GetInventoryBySkuAndLocation(sku, location)
		if existing != nil {
			result.Status = responses.InventoryStatusExists
			result.ExistingInventory = &responses.InventoryValidationMatch{
				ID:       existing.ID,
				SKU:      existing.SKU,
				Name:     existing.Name,
				Location: existing.Location,
				Quantity: existing.Quantity,
			}
			results = append(results, result)
			continue
		}

		// Same SKU at different location (similar)
		all, _ := r.GetAllInventory()
		var similar []responses.InventoryValidationMatch
		for _, inv := range all {
			if strings.EqualFold(inv.SKU, sku) && !strings.EqualFold(inv.Location, location) {
				similar = append(similar, responses.InventoryValidationMatch{
					ID: inv.ID, SKU: inv.SKU, Name: inv.Name, Location: inv.Location, Quantity: inv.Quantity,
				})
				if len(similar) == 3 {
					break
				}
			}
		}
		if len(similar) > 0 {
			result.Status = responses.InventoryStatusSimilar
			result.SimilarInventory = similar
			results = append(results, result)
			continue
		}

		result.Status = responses.InventoryStatusNew
		results = append(results, result)
	}
	return results, nil
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

// GetPickSuggestionsBySKU returns all (location, lot, quantity) rows for the given SKU.
// Caller (service) is responsible for sorting by rotation (FIFO/FEFO) then by quantity ascending.
func (r *InventoryRepository) GetPickSuggestionsBySKU(sku string) ([]dto.PickSuggestion, *responses.InternalResponse) {
	var rows []dto.PickSuggestion
	err := r.DB.
		Table("inventory_lots").
		Select("inv.location AS location, lots.id AS lot_id, lots.lot_number AS lot_number, inventory_lots.quantity AS quantity, lots.expiration_date AS expiration_date, lots.created_at AS lot_created_at").
		Joins("INNER JOIN inventory inv ON inventory_lots.inventory_id = inv.id").
		Joins("INNER JOIN lots ON inventory_lots.lot_id = lots.id").
		Where("inv.sku = ? AND inventory_lots.quantity > 0", sku).
		Scan(&rows).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener sugerencias de picking",
			Handled: false,
		}
	}
	return rows, nil
}

func (r *InventoryRepository) GetInventoryLots(inventoryID string) ([]responses.InventoryLot, *responses.InternalResponse) {
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

func (r *InventoryRepository) GetInventorySerials(inventoryID string) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) {
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

func (r *InventoryRepository) CreateInventoryLot(id string, input *requests.CreateInventoryLotRequest) *responses.InternalResponse {
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

func (r *InventoryRepository) DeleteInventoryLot(id string) *responses.InternalResponse {
	if err := r.DB.Where("id = ?", id).Delete(&database.InventoryLot{}).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar lote de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) CreateInventorySerial(id string, input *requests.CreateInventorySerial) *responses.InternalResponse {
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

func (r *InventoryRepository) DeleteInventorySerial(id string) *responses.InternalResponse {
	if err := r.DB.Where("id = ?", id).Delete(&database.InventorySerial{}).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar número de serie de inventario",
			Handled: false,
		}
	}

	return nil
}

func (r *InventoryRepository) GenerateImportTemplate(language string) ([]byte, error) {
	l2 := getLang(language)
	isEs := language != "en"
	title := "Importar Inventario"; subtitle := "Plantilla de importación — eSTOCK"
	instrTitle := "📋 Instrucciones"; instrContent := "1. Complete desde la fila 9  •  2. SKU, Nombre, Ubicación y Cantidad son obligatorios (*)  •  3. Use Si/No para campos de rastreo  •  4. Lotes y seriales: separe con comas"
	if !isEs {
		title = "Import Inventory"; subtitle = "Inventory import template — eSTOCK"
		instrTitle = "📋 Instructions"; instrContent = "1. Fill in data from row 9  •  2. SKU, Name, Location and Quantity are required (*)  •  3. Use Yes/No for tracking fields  •  4. Lots and serials: separate with commas"
	}
	yes, no := l2["yes"], l2["no"]

	// Get unique presentations from articles for the dropdown
	var presentations []string
	r.DB.Table("articles").Distinct("presentation").Pluck("presentation", &presentations)
	if len(presentations) == 0 {
		presentations = []string{"unidad", "caja", "pallet", "paquete"}
	}

	cfg := ModuleTemplateConfig{
		DataSheetName: func() string { if isEs { return "Inventario" }; return "Inventory" }(),
		OptSheetName:  func() string { if isEs { return "Opciones" }; return "Options" }(),
		LogoOffsetX:   0,
		LogoOffsetY:   5,
		LogoScaleX:    0.108,
		LogoScaleY:    0.246,
		LogoAnchor:    "E1",
		Title: title, Subtitle: subtitle, InstrTitle: instrTitle, InstrContent: instrContent,
		Columns: func() []ColumnDef {
			if isEs {
				return []ColumnDef{
					{Header: "SKU *", Required: true, Width: 14},
					{Header: "Nombre *", Required: true, Width: 28},
					{Header: "Descripción", Required: false, Width: 28},
					{Header: "Ubicación *", Required: true, Width: 18},
					{Header: "Cantidad *", Required: true, Width: 12},
					{Header: "Precio unitario", Required: false, Width: 14},
					{Header: "Rastrear por lote", Required: false, Width: 16},
					{Header: "Rastrear por serie", Required: false, Width: 16},
					{Header: "Rastrear expiración", Required: false, Width: 18},
					{Header: "Cantidad Mínima", Required: false, Width: 14},
					{Header: "Cantidad Máxima", Required: false, Width: 14},
				}
			}
			return []ColumnDef{
				{Header: "SKU *", Required: true, Width: 14},
				{Header: "Name *", Required: true, Width: 28},
				{Header: "Description", Required: false, Width: 28},
				{Header: "Location *", Required: true, Width: 18},
				{Header: "Quantity *", Required: true, Width: 12},
				{Header: "Unit Price", Required: false, Width: 14},
				{Header: "Track by Lot", Required: false, Width: 16},
				{Header: "Track by Serial", Required: false, Width: 16},
				{Header: "Track Expiration", Required: false, Width: 18},
				{Header: "Min Quantity", Required: false, Width: 14},
				{Header: "Max Quantity", Required: false, Width: 14},
			}
		}(),
		ExampleRow: []string{"SKU-0001", func() string { if isEs { return "Producto Ejemplo" }; return "Example Product" }(), "", "LOC-001", "100", "9.99", yes, no, no, "5", "500"},
		ApplyValidations: func(f *excelize.File, dataSheet, optSheet string, start, end int) error {
			f.NewSheet(optSheet)
			// Col A: presentations; Col B: yes/no
			for i, v := range presentations { cell, _ := excelize.CoordinatesToCellName(1, i+1); f.SetCellValue(optSheet, cell, v) }
			f.SetCellValue(optSheet, "B1", yes); f.SetCellValue(optSheet, "B2", no)
			f.SetSheetVisible(optSheet, false)
			presRef := "'" + optSheet + "'!$A$1:$A$" + fmt.Sprintf("%d", len(presentations))
			yesNoRef := "'" + optSheet + "'!$B$1:$B$2"
			errPres := func() string { if isEs { return "Presentación inválida" }; return "Invalid presentation" }()
			errBool := func() string { if isEs { return "Valor inválido" }; return "Invalid value" }()
			errNum := func() string { if isEs { return "Cantidad inválida" }; return "Invalid quantity" }()
			// No presentation column in inventory template; apply yes/no to cols G,H,I
			if err := addDropListValidation(f, dataSheet, "G9:I2000", yesNoRef, errBool, errBool); err != nil { return err }
			_ = presRef; _ = errPres
			return addNumericMinValidation(f, dataSheet, "E9:E2000", excelize.DataValidationTypeDecimal, errNum, errNum)
		},
	}
	return BuildModuleImportTemplate(cfg)
}
