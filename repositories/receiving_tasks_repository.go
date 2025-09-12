package repositories

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReceivingTasksRepository struct {
	DB *gorm.DB
}

func (r *ReceivingTasksRepository) GetAllReceivingTasks() ([]database.ReceivingTask, *responses.InternalResponse) {
	var tasks []database.ReceivingTask

	err := r.DB.
		Table(database.ReceivingTask{}.TableName()).
		Order("created_at DESC").
		Find(&tasks).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch receiving tasks",
			Handled: false,
		}
	}

	return tasks, nil
}

func (r *ReceivingTasksRepository) GetReceivingTaskByID(id int) (*database.ReceivingTask, *responses.InternalResponse) {
	var task database.ReceivingTask

	err := r.DB.
		Table(database.ReceivingTask{}.TableName()).
		Where("id = ?", id).
		First(&task).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Receiving task not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch receiving task",
			Handled: false,
		}
	}

	return &task, nil
}

func (r *ReceivingTasksRepository) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	var items []requests.ReceivingTaskItemRequest

	if err := json.Unmarshal(task.Items, &items); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
	}

	nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
	taskID := fmt.Sprintf("RCV-%06d", nowMillis%1_000_000)

	priority := task.Priority
	if priority == "" {
		priority = "normal"
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validate items
		articleCache := make(map[string]database.Article)

		for idx := range items {
			sku := items[idx].SKU

			// Resolve article (cached)
			art, ok := articleCache[sku]
			if !ok {
				if err := tx.Where("sku = ?", sku).First(&art).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return fmt.Errorf("article not found for SKU %s", sku)
					}
					return fmt.Errorf("find article %s: %w", sku, err)
				}
				articleCache[sku] = art
			}

			expected := float64(items[idx].ExpectedQuantity)

			// If tracked by lot: sum of lots must equal expected
			if art.TrackByLot {
				if len(items[idx].LotNumbers) == 0 {
					return fmt.Errorf("SKU %s is lot-tracked: lots are required", sku)
				}

				var sumLots float64
				seen := make(map[string]bool) // optional: avoid duplicates in the same payload
				for _, ln := range items[idx].LotNumbers {
					if ln.LotNumber == "" {
						return fmt.Errorf("SKU %s: lot_number is required for lot-tracked items", sku)
					}
					if seen[ln.LotNumber] {
						return fmt.Errorf("SKU %s: duplicated lot_number '%s' in payload", sku, ln.LotNumber)
					}
					seen[ln.LotNumber] = true
					sumLots += ln.Quantity
				}

				if math.Abs(sumLots-expected) > 1e-9 {
					// The sum of lot quantities must match expected quantity
					return fmt.Errorf("The sum of lot quantities does not match expected_qty")
				}

				items[idx].Status = tools.StrPtr("open")
				for i := 0; i < len(items[idx].LotNumbers); i++ {
					items[idx].LotNumbers[i].Status = tools.StrPtr("open")
				}
			}

			// If tracked by serial: count of serials must equal expected
			if art.TrackBySerial {
				if len(items[idx].SerialNumbers) == 0 {
					return fmt.Errorf("SKU %s is serial-tracked: serials[] are required", sku)
				}
				if float64(len(items[idx].SerialNumbers)) != expected {
					return fmt.Errorf("SKU %s: len(serials)=%d does not match expected_qty=%.0f",
						sku, len(items[idx].SerialNumbers), expected)
				}
			}
		}

		// 2) Create receiving task
		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal items: %w", err)
		}

		receivingTask := database.ReceivingTask{
			TaskID:        taskID,
			InboundNumber: task.InboundNumber,
			CreatedBy:     userId,
			AssignedTo:    task.AssignedTo,
			Status:        "open",
			Priority:      priority,
			Notes:         task.Notes,
			Items:         itemsJSON,
		}

		if err := tx.Create(&receivingTask).Error; err != nil {
			return fmt.Errorf("create task: %w", err)
		}

		// 3) Upsert lots/serials to keep catalogs consistent
		for i := 0; i < len(items); i++ {
			sku := items[i].SKU
			art := articleCache[sku]

			// Lots
			if art.TrackByLot && len(items[i].LotNumbers) > 0 {
				for j := 0; j < len(items[i].LotNumbers); j++ {
					parsedDate := (*time.Time)(nil)
					if items[i].LotNumbers[j].ExpirationDate != nil {
						parsedDate = tools.ParseDate(*items[i].LotNumbers[j].ExpirationDate)
					}

					lot := database.Lot{
						LotNumber:      items[i].LotNumbers[j].LotNumber,
						SKU:            sku,
						Quantity:       items[i].LotNumbers[j].Quantity,
						CreatedAt:      tools.GetCurrentTime(),
						UpdatedAt:      tools.GetCurrentTime(),
						ExpirationDate: parsedDate,
					}

					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&lot).Error; err != nil {
						return fmt.Errorf("create lot %s/%s: %w", sku, lot.LotNumber, err)
					}
				}
			}

			// Serials
			if art.TrackBySerial && len(items[i].SerialNumbers) > 0 {
				for j := 0; j < len(items[i].SerialNumbers); j++ {
					serial := database.Serial{
						SerialNumber: items[i].SerialNumbers[j].SerialNumber,
						SKU:          sku,
						Status:       "available",
						CreatedAt:    tools.GetCurrentTime(),
						UpdatedAt:    tools.GetCurrentTime(),
					}

					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&serial).Error; err != nil {
						return fmt.Errorf("create serial %s: %w", serial.SerialNumber, err)
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		// Return validation detail in the message
		return &responses.InternalResponse{Error: err, Message: err.Error(), Handled: true}
	}

	return nil
}

func (r *ReceivingTasksRepository) UpdateReceivingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.ReceivingTask
		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Receiving task not found", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to retrieve receiving task"}
			return nil
		}

		protected := map[string]bool{
			"id":         true,
			"task_id":    true,
			"created_at": true,
		}

		whitelist := map[string]bool{
			"assigned_to":    true,
			"priority":       true,
			"status":         true,
			"notes":          true,
			"items":          true,
			"inbound_number": true,
			"updated_at":     true,
			"completed_at":   true,
		}

		clean := make(map[string]interface{}, len(data)+2)
		for k, v := range data {
			key := strings.ToLower(k)
			key = strings.ReplaceAll(key, "inboundnumber", "inbound_number")
			key = strings.ReplaceAll(key, "assignedto", "assigned_to")
			key = strings.ReplaceAll(key, "completedat", "completed_at")
			key = strings.ReplaceAll(key, "updatedat", "updated_at")

			if protected[key] {
				continue
			}
			if !whitelist[key] {
				continue
			}
			clean[key] = v
		}

		clean["updated_at"] = tools.GetCurrentTime()

		if raw, ok := clean["status"]; ok {
			if s, ok := raw.(string); ok {
				sLower := strings.ToLower(strings.TrimSpace(s))
				switch sLower {
				case "completed":
					clean["completed_at"] = tools.GetCurrentTime()
				default:
					clean["completed_at"] = gorm.Expr("NULL")
				}
				clean["status"] = sLower
			}
		}

		if items, ok := clean["items"]; ok {
			switch it := items.(type) {
			case map[string]interface{}, []interface{}:
				b, err := json.Marshal(it)
				if err != nil {
					*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
					return nil
				}
				clean["items"] = b
			}
		}

		if err := r.DB.Model(&task).Updates(clean).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
			return nil
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Transaction failed"}
	}

	return nil
}

func (r *ReceivingTasksRepository) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to open Excel file"}
	}
	defer f.Close()

	const sheet = "Sheet1"

	rows, err := f.GetRows(sheet)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to read rows"}
	}
	if len(rows) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("empty sheet"), Message: "Excel has no data", Handled: true}
	}

	getLabeledValue := func(label string) *string {
		lab := strings.ToLower(strings.TrimSpace(label))
		for _, row := range rows[:minInt(len(rows), 30)] {
			for j, cell := range row {
				if strings.EqualFold(strings.TrimSpace(cell), lab) {
					for k := j + 1; k < len(row); k++ {
						val := strings.TrimSpace(row[k])
						if val != "" {
							return ptr(val)
						}
					}
					return ptr("")
				}
			}
		}
		return nil
	}

	headerRowIdx := -1
	colIndex := map[string]int{}

	for i, row := range rows {
		found := 0
		tmp := map[string]int{}
		for j, cell := range row {
			key := strings.ToLower(strings.TrimSpace(cell))
			switch key {
			case "sku":
				tmp["sku"] = j
				found++
			case "expected quantity":
				tmp["expected quantity"] = j
				found++
			case "location":
				tmp["location"] = j
				found++
			case "lot numbers":
				tmp["lot numbers"] = j
				found++
			case "serial numbers":
				tmp["serial numbers"] = j
				found++
			}
		}
		if found >= 3 && tmp["sku"] >= 0 {
			headerRowIdx = i
			colIndex = tmp
			break
		}
	}
	if headerRowIdx == -1 {
		return &responses.InternalResponse{Error: fmt.Errorf("headers not found"), Message: "Items header row not found (SKU, Expected Quantity...)", Handled: true}
	}

	inboundNumber := getLabeledValue("Inbound Number")
	assignedTo := getLabeledValue("Assigned To")
	priority := getLabeledValue("Priority")
	notes := getLabeledValue("Notes")

	priorityNorm := "normal"
	if priority != nil && strings.TrimSpace(*priority) != "" {
		p := strings.ToLower(strings.TrimSpace(*priority))
		switch p {
		case "low", "baja":
			priorityNorm = "low"
		case "high", "alta":
			priorityNorm = "high"
		default:
			priorityNorm = "normal"
		}
	}

	var items []database.ReceivingTaskItem

	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]
		sku := get(row, colIndex["sku"])
		if strings.TrimSpace(sku) == "" {
			continue
		}

		expQtyStr := get(row, colIndex["expected quantity"])
		location := get(row, colIndex["location"])
		lotsStr := get(row, colIndex["lot numbers"])
		serialsStr := get(row, colIndex["serial numbers"])

		expQty := 0
		if n, err := strconv.Atoi(strings.TrimSpace(expQtyStr)); err == nil {
			expQty = n
		}

		lots := splitCSV(lotsStr)
		serials := splitCSV(serialsStr)

		items = append(items, database.ReceivingTaskItem{
			SKU:              strings.TrimSpace(sku),
			ExpectedQuantity: expQty,
			Location:         strings.TrimSpace(location),
			LotNumbers:       lots,
			SerialNumbers:    serials,
		})
	}
	if len(items) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("no items"), Message: "No items found to import", Handled: true}
	}

	itemsJSON, _ := json.Marshal(items)
	req := &requests.CreateReceivingTaskRequest{
		InboundNumber: safeDeref(inboundNumber),
		AssignedTo:    assignedTo,
		Priority:      priorityNorm,
		Notes:         notes,
		Items:         json.RawMessage(itemsJSON),
	}

	if resp := r.CreateReceivingTask(userID, req); resp != nil && resp.Error != nil {
		return resp
	}
	return &responses.InternalResponse{
		Message: "Receiving task imported and created successfully",
		Handled: true,
	}
}

func (r *ReceivingTasksRepository) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	tasks, errResp := r.GetAllReceivingTasks()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Task ID", "Inbound Number", "Created By", "Assigned To", "Status", "Priority", "Notes", "Items", "Created At", "Updated At", "Completed At"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, task := range tasks {
		rowNum := i + 2
		row := []interface{}{
			task.ID,
			task.TaskID,
			task.InboundNumber,
			task.CreatedBy,
			task.AssignedTo,
			task.Status,
			task.Priority,
			task.Notes,
			string(task.Items),
			task.CreatedAt.Format(time.RFC3339),
			nil,
			nil,
		}
		if !task.UpdatedAt.IsZero() {
			row[10] = task.UpdatedAt.Format(time.RFC3339)
		}
		if !task.CompletedAt.IsZero() {
			row[11] = task.CompletedAt.Format(time.RFC3339)
		}

		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			f.SetCellValue(sheet, cell, val)
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

func (r *ReceivingTasksRepository) CompleteFullTask(id int, location string) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Get the task
		var task database.ReceivingTask

		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Receiving task not found", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to retrieve receiving task"}
			return nil
		}

		// Process items
		var items []requests.ReceivingTaskItemRequest
		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			return nil
		}

		// Create inventory
		for i := 0; i < len(items); i++ {
			sku := items[i].SKU

			items[i].Status = tools.StrPtr("completed")
			items[i].ReceivedQuantity = tools.IntToPtr(int(items[i].ExpectedQuantity))

			var article database.Article
			if err := tx.Where("sku = ?", sku).First(&article).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				return fmt.Errorf("find article %s: %w", sku, err)
			}

			var inventory database.Inventory

			inventory.SKU = sku
			inventory.Name = article.Name
			inventory.Description = article.Description
			inventory.Location = location
			inventory.Quantity = tools.IntToFloat64(items[i].ExpectedQuantity)
			inventory.Status = "available"
			inventory.Presentation = article.Presentation
			inventory.UnitPrice = article.UnitPrice
			inventory.CreatedAt = time.Now()
			inventory.UpdatedAt = time.Now()

			if err := tx.Create(&inventory).Error; err != nil {
				return errors.New("Failed to create inventory")
			}

			if article.TrackByLot && items[i].LotNumbers != nil {
				for j := 0; j < len(items[i].LotNumbers); j++ {
					lotNum := items[i].LotNumbers[j]

					var lot database.Lot

					if err := tx.Where("lot_number = ? AND sku = ?", lotNum.LotNumber, items[i].SKU).First(&lot).Error; err != nil {
						return errors.New("Failed to retrieve existing lot")
					}

					// Update lot status to available
					lot.Status = tools.StrPtr("available")
					lot.UpdatedAt = tools.GetCurrentTime()

					if err := tx.Save(&lot).Error; err != nil {
						return errors.New("Failed to update lot status")
					}

					inventoryLot := &database.InventoryLot{
						InventoryID: inventory.ID,
						LotID:       lot.ID,
						Quantity:    lotNum.Quantity,
						Location:    items[i].Location,
					}

					if err := tx.Create(inventoryLot).Error; err != nil {
						return errors.New("Failed to create inventory_lot association")
					}

					// Set lotNum.Status to completed
					items[i].LotNumbers[j].Status = tools.StrPtr("completed")
					items[i].LotNumbers[j].ReceivedQuantity = &lotNum.Quantity
				}
			}

			if article.TrackBySerial && items[i].SerialNumbers != nil {
				for k := 0; k < len(items[i].SerialNumbers); k++ {
					serial := items[i].SerialNumbers[k]

					var serialCount int64
					err := tx.Model(&database.Serial{}).
						Where("serial_number = ? AND sku = ?", serial.SerialNumber, items[i].SKU).
						Count(&serialCount).Error
					if err != nil {
						return errors.New("Failed to check existing serial")
					}

					if serialCount == 0 {
						newSerial := &database.Serial{
							SerialNumber: serial.SerialNumber,
							SKU:          items[i].SKU,
							CreatedAt:    tools.GetCurrentTime(),
							UpdatedAt:    tools.GetCurrentTime(),
							Status:       "available",
						}
						if err := tx.Create(newSerial).Error; err != nil {
							return errors.New("Failed to create serial")
						}

						inventorySerial := &database.InventorySerial{
							InventoryID: inventory.ID,
							SerialID:    newSerial.ID,
							Location:    items[i].Location,
						}
						if err := tx.Create(inventorySerial).Error; err != nil {
							return errors.New("Failed to create inventory_serial association")
						}
					}
				}
			}
		}

		// Update items
		updatedItems, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal updated items: %w", err)
		}

		task.Items = updatedItems

		// Update task fields
		clean := map[string]interface{}{
			"items":        updatedItems,
			"status":       "completed",
			"completed_at": tools.GetCurrentTime(),
			"updated_at":   tools.GetCurrentTime(),
		}

		if task.Status == "completed" {
			*handledResp = responses.InternalResponse{Message: "Receiving task is already completed", Handled: true}
			return nil
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
			return nil
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Transaction failed"}
	}

	if handledResp.Error != nil || handledResp.Handled {
		return handledResp
	}

	return nil
}

func (r *ReceivingTasksRepository) CompleteReceivingLine(id int, location string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.ReceivingTask

		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Receiving task not found", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to retrieve receiving task"}
			return nil
		}

		var items []requests.ReceivingTaskItemRequest
		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			return nil
		}

		found := false
		for i := 0; i < len(items); i++ {
			if items[i].SKU == item.SKU {
				found = true
				break
			}
		}

		if !found {
			*handledResp = responses.InternalResponse{Message: "SKU not found in task items", Handled: true}
			return nil
		}

		var article database.Article

		if err := tx.Where("sku = ?", item.SKU).First(&article).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				*handledResp = responses.InternalResponse{Message: "Article not found for SKU", Handled: true}
				return nil
			}
			return fmt.Errorf("find article %s: %w", item.SKU, err)
		}

		var qty float64
		if article.TrackByLot && item.LotNumbers != nil {
			for _, lot := range item.LotNumbers {
				qty += lot.Quantity
			}
		} else if article.TrackBySerial && item.SerialNumbers != nil {
			qty = float64(len(item.SerialNumbers))
		} else {
			qty = tools.IntToFloat64(item.ExpectedQuantity)
		}

		if qty <= 0 || qty < float64(item.ExpectedQuantity) {
			// Update items status to partial for this SKU
			for i := 0; i < len(items); i++ {
				if items[i].SKU == item.SKU {
					items[i].Status = tools.StrPtr("partial")
					items[i].ReceivedQuantity = tools.IntToPtr(int(qty))
					break
				}
			}

			updatedItems, err := json.Marshal(items)
			if err != nil {
				return fmt.Errorf("marshal updated items: %w", err)
			}

			task.Items = updatedItems

			clean := map[string]interface{}{
				"items":      updatedItems,
				"updated_at": tools.GetCurrentTime(),
			}

			if err := tx.Model(&task).Updates(clean).Error; err != nil {
				*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
				return nil
			}
		}

		var inventory database.Inventory

		inventory.SKU = item.SKU
		inventory.Name = article.Name
		inventory.Description = article.Description
		inventory.Location = location
		inventory.Quantity = qty
		inventory.Status = "available"
		inventory.Presentation = article.Presentation
		inventory.UnitPrice = article.UnitPrice
		inventory.CreatedAt = time.Now()
		inventory.UpdatedAt = time.Now()

		if err := tx.Create(&inventory).Error; err != nil {
			return errors.New("Failed to create inventory")
		}

		if article.TrackByLot && item.LotNumbers != nil {
			for j := 0; j < len(item.LotNumbers); j++ {
				lotNum := item.LotNumbers[j]

				var lotCount int64
				err := tx.Model(&database.Lot{}).
					Where("lot_number = ? AND sku = ?", lotNum.LotNumber, item.SKU).
					Count(&lotCount).Error
				if err != nil {
					return errors.New("Failed to check existing lot")
				}

				lotId := 0

				if lotCount == 0 {
					var expirationDate *time.Time
					if lotNum.ExpirationDate != nil {
						parsed, _ := time.Parse("2006-01-02", *lotNum.ExpirationDate)
						expirationDate = &parsed
					}

					lot := &database.Lot{
						LotNumber:      lotNum.LotNumber,
						SKU:            item.SKU,
						Quantity:       lotNum.Quantity,
						ExpirationDate: expirationDate,
						CreatedAt:      tools.GetCurrentTime(),
						UpdatedAt:      tools.GetCurrentTime(),
					}

					if err := tx.Create(lot).Error; err != nil {
						return errors.New("Failed to create lot")
					}

					lotId = lot.ID

				} else {
					var lot database.Lot
					if err := tx.Where("lot_number = ? AND sku = ?", lotNum.LotNumber, item.SKU).First(&lot).Error; err != nil {
						return errors.New("Failed to retrieve existing lot")
					}

					if lot.Quantity != item.LotNumbers[j].Quantity {
						// Update items lot number position status to partial for this SKU
						for i := 0; i < len(items); i++ {
							if items[i].SKU == item.SKU {
								for k := 0; k < len(items[i].LotNumbers); k++ {
									if items[i].LotNumbers[k].LotNumber == lot.LotNumber {
										items[i].LotNumbers[k].Status = tools.StrPtr("partial")
										items[i].LotNumbers[k].ReceivedQuantity = &lotNum.Quantity
										break
									}
								}
								items[i].Status = tools.StrPtr("partial")
								break
							}
						}

						updatedItems, err := json.Marshal(items)

						if err != nil {
							return fmt.Errorf("marshal updated items: %w", err)
						}

						task.Items = updatedItems

						clean := map[string]interface{}{
							"items":      updatedItems,
							"updated_at": tools.GetCurrentTime(),
						}

						if err := tx.Model(&task).Updates(clean).Error; err != nil {
							*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
							return nil
						}
					} else {
						for i := 0; i < len(items); i++ {
							if items[i].SKU == item.SKU {
								for k := 0; k < len(items[i].LotNumbers); k++ {
									if items[i].LotNumbers[k].LotNumber == lot.LotNumber {
										items[i].LotNumbers[k].Status = tools.StrPtr("completed")
										items[i].LotNumbers[k].ReceivedQuantity = &lotNum.Quantity
										break
									}
								}
								items[i].Status = tools.StrPtr("completed")
								break
							}
						}

						updatedItems, err := json.Marshal(items)

						if err != nil {
							return fmt.Errorf("marshal updated items: %w", err)
						}

						task.Items = updatedItems

						clean := map[string]interface{}{
							"items":      updatedItems,
							"updated_at": tools.GetCurrentTime(),
						}

						if err := tx.Model(&task).Updates(clean).Error; err != nil {
							*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
							return nil
						}

						// Update lot status to available
						lot.Status = tools.StrPtr("available")
						lot.UpdatedAt = tools.GetCurrentTime()

						if err := tx.Save(&lot).Error; err != nil {
							return errors.New("Failed to update lot status")
						}
					}

					lotId = lot.ID
				}

				inventoryLot := &database.InventoryLot{
					InventoryID: inventory.ID,
					LotID:       lotId,
					Quantity:    lotNum.Quantity,
					Location:    item.Location,
				}

				if err := tx.Create(inventoryLot).Error; err != nil {
					return errors.New("Failed to create inventory_lot association")
				}
			}
		}

		if article.TrackBySerial && item.SerialNumbers != nil {
			for k := 0; k < len(item.SerialNumbers); k++ {
				serial := item.SerialNumbers[k]

				var serialCount int64
				err := tx.Model(&database.Serial{}).
					Where("serial_number = ? AND sku = ?", serial.SerialNumber, item.SKU).
					Count(&serialCount).Error
				if err != nil {
					return errors.New("Failed to check existing serial")
				}

				if serialCount == 0 {
					newSerial := &database.Serial{
						SerialNumber: serial.SerialNumber,
						SKU:          item.SKU,
						CreatedAt:    tools.GetCurrentTime(),
						UpdatedAt:    tools.GetCurrentTime(),
						Status:       "available",
					}
					if err := tx.Create(newSerial).Error; err != nil {
						return errors.New("Failed to create serial")
					}

					inventorySerial := &database.InventorySerial{
						InventoryID: inventory.ID,
						SerialID:    newSerial.ID,
						Location:    item.Location,
					}
					if err := tx.Create(inventorySerial).Error; err != nil {
						return errors.New("Failed to create inventory_serial association")
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Transaction failed"}
	}

	if handledResp.Error != nil || handledResp.Handled {
		return handledResp
	}

	return nil
}
