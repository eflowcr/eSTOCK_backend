package repositories

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type PickingTaskRepository struct {
	DB *gorm.DB
}

func (r *PickingTaskRepository) GetAllPickingTasks() ([]database.PickingTask, *responses.InternalResponse) {
	var tasks []database.PickingTask

	err := r.DB.Table(database.PickingTask{}.TableName()).Order("created_at DESC").Find(&tasks).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch picking tasks",
			Handled: false,
		}
	}

	return tasks, nil
}

func (r *PickingTaskRepository) GetPickingTaskByID(id int) (*database.PickingTask, *responses.InternalResponse) {
	var task database.PickingTask

	err := r.DB.Table(database.PickingTask{}.TableName()).Where("id = ?", id).First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Picking task not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch picking task",
			Handled: false,
		}
	}

	return &task, nil
}

func (r *PickingTaskRepository) CreatePickingTask(userId string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	var items []requests.PickingTaskItemRequest
	if err := json.Unmarshal(task.Items, &items); err != nil {
		*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
		return handledResp
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
		taskID := fmt.Sprintf("PICK-%06d", nowMillis%1_000_000)

		articleCache := make(map[string]database.Article)

		for i := range items {
			// Asignar status inicial una sola vez
			items[i].Status = tools.StrPtr("open")
			sku := items[i].SKU

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

			if art.TrackByLot {
				for j := range items[i].LotNumbers {
					items[i].LotNumbers[j].Status = tools.StrPtr("open")
				}
			}

			if art.TrackBySerial {
				for j := range items[i].SerialNumbers {
					// ⚠️ Esto depende del tipo de Status
					items[i].SerialNumbers[j].Status = *tools.StrPtr("open")
				}
			}
		}

		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal items: %w", err)
		}

		priority := task.Priority
		if priority == "" {
			priority = "normal"
		}

		pickingTask := database.PickingTask{
			TaskID:      taskID,
			OrderNumber: task.OutboundNumber,
			CreatedBy:   userId,
			AssignedTo:  task.AssignedTo,
			Status:      "open",
			Priority:    priority,
			Notes:       task.Notes,
			Items:       json.RawMessage(itemsJSON),
		}

		if err := tx.Create(&pickingTask).Error; err != nil {
			return fmt.Errorf("create picking task: %w", err)
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

func (r *PickingTaskRepository) UpdatePickingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	var task database.PickingTask
	if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{Message: "Picking task not found", Handled: true}
		}
		return &responses.InternalResponse{Error: err, Message: "Failed to retrieve picking task"}
	}

	protected := map[string]bool{
		"id":         true,
		"task_id":    true,
		"created_at": true,
	}

	whitelist := map[string]bool{
		"assigned_to":  true,
		"priority":     true,
		"status":       true,
		"notes":        true,
		"items":        true,
		"order_number": true,
		"updated_at":   true,
		"completed_at": true,
	}

	clean := make(map[string]interface{}, len(data)+2)
	for k, v := range data {
		key := strings.ToLower(k)
		key = strings.ReplaceAll(key, "assignedto", "assigned_to")
		key = strings.ReplaceAll(key, "ordernumber", "order_number")
		key = strings.ReplaceAll(key, "outboundnumber", "order_number")
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
				return &responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			}
			clean["items"] = b
		}
	}

	if err := r.DB.Model(&task).Updates(clean).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to update picking task"}
	}

	return nil
}

func (r *PickingTaskRepository) ImportPickingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
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

	getOneOf := func(labels ...string) *string {
		for _, row := range rows[:minInt(len(rows), 30)] {
			for j, cell := range row {
				cellNorm := strings.ToLower(strings.TrimSpace(cell))
				for _, lab := range labels {
					if cellNorm == strings.ToLower(strings.TrimSpace(lab)) {
						for k := j + 1; k < len(row); k++ {
							if v := strings.TrimSpace(row[k]); v != "" {
								return ptr(v)
							}
						}
						return ptr("")
					}
				}
			}
		}
		return nil
	}

	outboundNumber := getOneOf("Outbound Number", "Order Number")
	assignedTo := getOneOf("Assigned To")
	priority := getOneOf("Priority")
	notes := getOneOf("Notes")

	priorityNorm := "normal"
	if priority != nil && strings.TrimSpace(*priority) != "" {
		switch p := strings.ToLower(strings.TrimSpace(*priority)); p {
		case "low", "baja":
			priorityNorm = "low"
		case "high", "alta":
			priorityNorm = "high"
		default:
			priorityNorm = "normal"
		}
	}

	headerRowIdx := -1
	colIndex := map[string]int{}

	for i, row := range rows {
		tmp := map[string]int{
			"sku":            -1,
			"qty":            -1,
			"location":       -1,
			"lot_numbers":    -1,
			"serial_numbers": -1,
		}
		found := 0
		for j, cell := range row {
			key := strings.ToLower(strings.TrimSpace(cell))
			switch key {
			case "sku":
				tmp["sku"] = j
				found++
			case "expected quantity", "requested quantity":
				tmp["qty"] = j
				found++
			case "location", "from location":
				tmp["location"] = j
				found++
			case "lot numbers":
				tmp["lot_numbers"] = j
				found++
			case "serial numbers":
				tmp["serial_numbers"] = j
				found++
			}
		}
		if tmp["sku"] >= 0 && (tmp["qty"] >= 0 || tmp["location"] >= 0) {
			headerRowIdx = i
			colIndex = tmp
			break
		}
	}
	if headerRowIdx == -1 {
		return &responses.InternalResponse{
			Error:   fmt.Errorf("headers not found"),
			Message: "Items header row not found (SKU, Expected/Requested Quantity, Location, Lot Numbers, Serial Numbers)",
			Handled: true,
		}
	}

	var items []database.PickingTaskItem

	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]

		sku := get(row, colIndex["sku"])
		if strings.TrimSpace(sku) == "" {
			continue
		}

		qtyStr := get(row, colIndex["qty"])
		location := get(row, colIndex["location"])
		lotsStr := get(row, colIndex["lot_numbers"])
		serialsStr := get(row, colIndex["serial_numbers"])

		qty := 0
		if n, err := strconv.Atoi(strings.TrimSpace(qtyStr)); err == nil {
			qty = n
		}

		lots := splitCSV(lotsStr)
		serials := splitCSV(serialsStr)

		items = append(items, database.PickingTaskItem{
			SKU:              strings.TrimSpace(sku),
			ExpectedQuantity: qty,
			Location:         strings.TrimSpace(location),
			LotNumbers:       lots,
			SerialNumbers:    serials,
		})
	}
	if len(items) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("no items"), Message: "No items found to import", Handled: true}
	}

	itemsJSON, _ := json.Marshal(items)
	req := &requests.CreatePickingTaskRequest{
		OutboundNumber: safeDeref(outboundNumber),
		AssignedTo:     assignedTo,
		Priority:       priorityNorm,
		Notes:          notes,
		Items:          json.RawMessage(itemsJSON),
	}

	if resp := r.CreatePickingTask(userID, req); resp != nil && resp.Error != nil {
		return resp
	}

	return &responses.InternalResponse{
		Message: "Picking task imported and created successfully",
		Handled: true,
	}
}

func (r *PickingTaskRepository) ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse) {
	tasks, errResp := r.GetAllPickingTasks()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID",
		"Task ID",
		"Order Number",
		"Created By",
		"Assigned To",
		"Status",
		"Priority",
		"Notes",
		"Items",
		"Created At",
		"Updated At",
		"Completed At",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, task := range tasks {
		rowNum := i + 2
		row := []interface{}{
			task.ID,
			task.TaskID,
			task.OrderNumber,
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

func (r *PickingTaskRepository) CompletePickingTask(id int, location, userId string) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Get the task
		var task database.PickingTask
		if err := tx.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Picking task not found", Handled: true}
				return nil
			}
			return fmt.Errorf("retrieve picking task: %w", err)
		}

		var items []requests.PickingTaskItemRequest

		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			return nil
		}

		for i := 0; i < len(items); i++ {
			sku := items[i].SKU

			items[i].Status = tools.StrPtr("completed")
			items[i].DeliveredQuantity = tools.IntToPtr(int(items[i].ExpectedQuantity))

			var article database.Article

			if err := tx.Where("sku = ?", sku).First(&article).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				return fmt.Errorf("find article %s: %w", sku, err)
			}

			var inventory database.Inventory

			// Check if there is enough stock in the specified location
			if err := tx.Where("sku = ? AND location = ?", sku, location).First(&inventory).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("not enough stock for SKU %s in location %s", sku, location)
				}
				return fmt.Errorf("find inventory %s in %s: %w", sku, location, err)
			}

			if inventory.Quantity < float64(items[i].ExpectedQuantity) {
				return fmt.Errorf("not enough stock for SKU %s in location %s", sku, location)
			}

			// Deduct the stock
			newQty := inventory.Quantity - float64(items[i].ExpectedQuantity)

			if err := tx.Model(&database.Inventory{}).Where("id = ?", inventory.ID).
				Update("quantity", newQty).Error; err != nil {
				return fmt.Errorf("update inventory %s in %s: %w", sku, location, err)
			}

			// Create inventory movement record
			movement := database.InventoryMovement{
				SKU:            sku,
				Location:       location,
				MovementType:   "picking",
				Quantity:       float64(items[i].ExpectedQuantity),
				RemainingStock: newQty,
				Reason:         tools.StrPtr(fmt.Sprintf("Picking Task %s", task.TaskID)),
				CreatedBy:      task.CreatedBy,
				CreatedAt:      tools.GetCurrentTime(),
			}

			if err := tx.Create(&movement).Error; err != nil {
				return fmt.Errorf("create inventory movement for %s in %s: %w", sku, location, err)
			}

			if article.TrackBySerial && items[i].SerialNumbers != nil {
				// Check if given serials count matches expected quantity
				if len(items[i].SerialNumbers) != items[i].ExpectedQuantity {
					// If not, then this task can't be completed fully
					*handledResp = responses.InternalResponse{Message: fmt.Sprintf("Serial numbers count (%d) does not match expected quantity (%d) for SKU %s", len(items[i].SerialNumbers), items[i].ExpectedQuantity, sku), Handled: true}
					return nil
				}

				for k := 0; k < len(items[i].SerialNumbers); k++ {
					serial := items[i].SerialNumbers[k]

					// Check if serial was created before
					var serialItem database.Serial

					// Check if serial exists and is in stock and its available
					if err := tx.Where("serial_number = ? AND sku = ? AND status = 'available'", serial.SerialNumber, sku).First(&serialItem).Error; err != nil {
						if err == gorm.ErrRecordNotFound {
							*handledResp = responses.InternalResponse{Message: fmt.Sprintf("Serial number %s for SKU %s not found in inventory", serial.SerialNumber, sku), Handled: true}
							return nil
						}
						return fmt.Errorf("find serial %s for SKU %s: %w", serial.SerialNumber, sku, err)
					}

					// Mark serial as picked
					serialItem.Status = "picked"
					serialItem.UpdatedAt = tools.GetCurrentTime()

					if err := tx.Save(&serialItem).Error; err != nil {
						return fmt.Errorf("update serial %s for SKU %s: %w", serial.SerialNumber, sku, err)
					}

					// Mark serial as completed in the task
					items[i].SerialNumbers[k].Status = "completed"
					items[i].SerialNumbers[k].ID = serialItem.ID
				}

				// Mark item as completed
				items[i].Status = tools.StrPtr("completed")
			}
		}

		updatedItems, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal updated items: %w", err)
		}

		task.Items = updatedItems

		clean := map[string]interface{}{
			"status":       "completed",
			"items":        updatedItems,
			"completed_at": tools.GetCurrentTime(),
			"updated_at":   tools.GetCurrentTime(),
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			return fmt.Errorf("update picking task: %w", err)
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

func (r *PickingTaskRepository) CompletePickingLine(id int, location, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.PickingTask

		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Picking task not found", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Failed to retrieve receiving task"}
			return nil
		}

		var items []requests.PickingTaskItemRequest
		var foundItem *requests.PickingTaskItemRequest

		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			return nil
		}

		found := false

		for i := 0; i < len(items); i++ {
			if items[i].SKU == item.SKU && items[i].Location == item.Location {
				foundItem = &items[i]
				found = true
				break
			}
		}

		if !found {
			*handledResp = responses.InternalResponse{Message: "Item not found in picking task", Handled: true}
			return nil
		}

		var article database.Article

		if err := tx.Where("sku = ?", foundItem.SKU).First(&article).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				*handledResp = responses.InternalResponse{Message: "Article not found for SKU " + foundItem.SKU, Handled: true}
				return nil
			}
			return fmt.Errorf("find article %s: %w", foundItem.SKU, err)
		}

		if foundItem.Status != nil && *foundItem.Status == "completed" {
			*handledResp = responses.InternalResponse{Message: "Item already completed", Handled: true}
			return nil
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

		if qty <= 0 || qty < float64(foundItem.ExpectedQuantity) {
			// Update items status to partial for this SKU
			for i := 0; i < len(items); i++ {
				if items[i].SKU == item.SKU {
					items[i].Status = tools.StrPtr("partial")
					items[i].DeliveredQuantity = tools.IntToPtr(int(qty))
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
				*handledResp = responses.InternalResponse{Error: err, Message: "Failed to update picking task"}
				return nil
			}
		} else {
			// If given quantity meets or exceeds expected, mark item as completed
			for i := 0; i < len(items); i++ {
				if items[i].SKU == item.SKU {
					items[i].Status = tools.StrPtr("completed")
					items[i].DeliveredQuantity = tools.IntToPtr(int(qty))
					break
				}
			}
		}

		var inventory database.Inventory

		// Check if there is enough stock in the specified location
		if err := tx.Where("sku = ? AND location = ?", item.SKU, location).First(&inventory).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("not enough stock for SKU %s in location %s", item.SKU, location)
			}
			return fmt.Errorf("find inventory %s in %s: %w", item.SKU, location, err)
		}

		if inventory.Quantity < qty {
			*handledResp = responses.InternalResponse{Message: fmt.Sprintf("Not enough stock for SKU %s in location %s", item.SKU, location), Handled: true}

			return nil
		}

		// Deduct the stock
		newQty := inventory.Quantity - qty

		if err := tx.Model(&database.Inventory{}).Where("id = ?", inventory.ID).
			Update("quantity", newQty).Error; err != nil {
			return fmt.Errorf("update inventory %s in %s: %w", item.SKU, location, err)
		}

		// Create inventory movement record
		movement := database.InventoryMovement{
			SKU:            item.SKU,
			Location:       location,
			MovementType:   "picking",
			Quantity:       qty,
			RemainingStock: newQty,
			Reason:         tools.StrPtr(fmt.Sprintf("Picking Task %s", task.TaskID)),
			CreatedBy:      task.CreatedBy,
			CreatedAt:      tools.GetCurrentTime(),
		}

		if err := tx.Create(&movement).Error; err != nil {
			return fmt.Errorf("create inventory movement for %s in %s: %w", item.SKU, location, err)
		}

		if article.TrackBySerial && item.SerialNumbers != nil {
			for k := 0; k < len(item.SerialNumbers); k++ {
				serial := item.SerialNumbers[k]

				// Check if serial is in stock and available
				var serialItem database.Serial
				if err := tx.Where("serial_number = ? AND sku = ? AND status = 'available'", serial.SerialNumber, item.SKU).First(&serialItem).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						*handledResp = responses.InternalResponse{Message: fmt.Sprintf("Serial number %s for SKU %s not found in inventory", serial.SerialNumber, item.SKU), Handled: true}
						return nil
					}
					return fmt.Errorf("find serial %s for SKU %s: %w", serial.SerialNumber, item.SKU, err)
				}

				// Mark serial as picked
				serialItem.Status = "picked"
				serialItem.UpdatedAt = tools.GetCurrentTime()

				if err := tx.Save(&serialItem).Error; err != nil {
					return fmt.Errorf("update serial %s for SKU %s: %w", serial.SerialNumber, item.SKU, err)
				}

				// Mark items serial as completed
				for i := 0; i < len(items); i++ {
					if items[i].SKU == item.SKU {
						for j := 0; j < len(items[i].SerialNumbers); j++ {
							if items[i].SerialNumbers[j].SerialNumber == serial.SerialNumber {
								items[i].SerialNumbers[j].Status = "completed"
								items[i].SerialNumbers[j].ID = serialItem.ID
								break
							}
						}
						break
					}
				}
			}

			if len(item.SerialNumbers) == foundItem.ExpectedQuantity {
				item.Status = tools.StrPtr("completed")
			} else {
				item.Status = tools.StrPtr("partial")
			}
		}

		updatedItems, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal updated items: %w", err)
		}

		task.Items = updatedItems

		clean := map[string]interface{}{
			"items":        updatedItems,
			"completed_at": tools.GetCurrentTime(),
			"updated_at":   tools.GetCurrentTime(),
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			return fmt.Errorf("update picking task: %w", err)
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
