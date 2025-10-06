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
	"gorm.io/gorm/clause"
)

type ReceivingTasksRepository struct {
	DB *gorm.DB
}

func (r *ReceivingTasksRepository) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	var tasks []responses.ReceivingTasksView

	sqlRaw := `
		SELECT
			rt.id,
			rt.task_id,
			rt.inbound_number,
			rt.created_by,
			usr.first_name || ' ' || usr.last_name AS created_by_name,
			rt.assigned_to,
			rt.status,
			rt.priority,
			rt.notes,
			rt.created_at,
			rt.updated_at,
			rt.completed_at,
			jsonb_agg(
				jsonb_build_object(
					'sku', item->>'sku',
					'item_name', a.name,
					'status', item->>'status',
					'location', item->>'location',
					'expected_qty', item->>'expected_qty',
					'received_qty', item->>'received_qty'					
				)
			) AS items
		FROM receiving_tasks rt
		INNER JOIN users usr ON rt.created_by = usr.id
		LEFT JOIN LATERAL jsonb_array_elements(rt.items) AS item ON TRUE
		LEFT JOIN articles a ON a.sku = item->>'sku'
		GROUP BY
			rt.id,
			rt.task_id,
			rt.inbound_number,
			rt.created_by,
			usr.first_name,
			usr.last_name,
			rt.assigned_to,
			rt.status,
			rt.priority,
			rt.notes,
			rt.created_at,
			rt.updated_at,
			rt.completed_at;
	`

	err := r.DB.Raw(sqlRaw).Scan(&tasks).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener todas las tareas de recepción",
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
				Message: "Tarea de recepción no encontrada",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la tarea de recepción",
			Handled: false,
		}
	}

	return &task, nil
}

func (r *ReceivingTasksRepository) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	var items []requests.ReceivingTaskItemRequest

	if err := json.Unmarshal(task.Items, &items); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Formato de items inválido", Handled: true}
	}

	nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
	taskID := fmt.Sprintf("RCV-%06d", nowMillis%1_000_000)

	priority := task.Priority
	if priority == "" {
		priority = "normal"
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Check if inbound number is unique
		var count int64

		if err := tx.Model(&database.ReceivingTask{}).Where("inbound_number = ?", task.InboundNumber).Count(&count).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al verificar la unicidad del número de entrada", Handled: false}

			return nil
		}

		if count > 0 {
			*handledResp = responses.InternalResponse{Error: fmt.Errorf("inbound number %s is already taken", task.InboundNumber), Message: "El número de entrada ya está en uso", Handled: true}
			return nil
		}

		// 1) Validate items
		articleCache := make(map[string]database.Article)

		for idx := range items {
			sku := items[idx].SKU
			items[idx].Status = tools.StrPtr("open")

			// Resolve article (cached)
			art, ok := articleCache[sku]
			if !ok {
				if err := tx.Where("sku = ?", sku).First(&art).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						*handledResp = responses.InternalResponse{Error: fmt.Errorf("article with SKU %s not found", sku), Message: fmt.Sprintf("Artículo con SKU %s no encontrado", sku), Handled: true}

						return nil
					}
					return fmt.Errorf("find article %s: %w", sku, err)
				}
				articleCache[sku] = art
			}

			// If tracked by lot: sum of lots must equal expected
			if art.TrackByLot {
				for i := 0; i < len(items[idx].LotNumbers); i++ {
					items[idx].LotNumbers[i].Status = tools.StrPtr("open")
				}
			}

			// If tracked by serial: count of serials must equal expected
			if art.TrackBySerial {
				for i := 0; i < len(items[idx].SerialNumbers); i++ {
					items[idx].SerialNumbers[i].Status = "open"
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
		return &responses.InternalResponse{Error: err, Message: "Error en la transacción"}
	}

	if handledResp.Error != nil || handledResp.Handled {
		return handledResp
	}

	return nil
}

func (r *ReceivingTasksRepository) UpdateReceivingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	var handledResp *responses.InternalResponse

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.ReceivingTask
		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Tarea de recepción no encontrada", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al obtener la tarea de recepción"}
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
					*handledResp = responses.InternalResponse{Error: err, Message: "Formato de items inválido", Handled: true}
					return nil
				}
				clean["items"] = b
			}
		}

		if err := r.DB.Model(&task).Updates(clean).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
			return nil
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error en la transacción"}
	}

	return nil
}

func (r *ReceivingTasksRepository) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al abrir el archivo de Excel"}
	}
	defer f.Close()

	const sheet = "Sheet1"

	rows, err := f.GetRows(sheet)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al leer las filas"}
	}
	if len(rows) == 0 {
		return &responses.InternalResponse{Error: fmt.Errorf("empty sheet"), Message: "El archivo de Excel no tiene datos", Handled: true}
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
		return &responses.InternalResponse{Error: fmt.Errorf("headers not found"), Message: "Fila de encabezado de items no encontrada (SKU, Cantidad Esperada...)", Handled: true}
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
		return &responses.InternalResponse{Error: fmt.Errorf("no items"), Message: "No se encontraron items para importar", Handled: true}
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
		Message: "Tarea de recepción importada exitosamente",
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
			Message: "Error al generar el archivo de Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func (r *ReceivingTasksRepository) CompleteFullTask(id int, location, userId string) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Get the task
		var task database.ReceivingTask

		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Tarea de recepción no encontrada", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al recuperar la tarea de recepción"}
			return nil
		}

		if task.Status == "closed" {
			*handledResp = responses.InternalResponse{Message: "La tarea de recepción ya está cerrada", Handled: true}
			return nil
		}

		// Process items
		var items []requests.ReceivingTaskItemRequest
		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Formato de items inválido", Handled: true}
			return nil
		}

		// Create inventory
		for i := 0; i < len(items); i++ {
			// Skip if item is already completed or closed
			if items[i].Status != nil && (*items[i].Status == "completed" || *items[i].Status == "partial") {
				continue
			}

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

			inventoryCount := int64(0)

			err := tx.Model(&database.Inventory{}).Where("sku = ? AND location = ?", sku, location).Count(&inventoryCount).Error

			if err != nil {
				return fmt.Errorf("check inventory for SKU %s and location %s: %w", sku, location, err)
			}

			if inventoryCount == 0 {
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
					return errors.New("failed to create inventory")
				}
			} else {
				if err := tx.First(&inventory, "sku = ? AND location = ?", sku, location).Error; err != nil {
					return fmt.Errorf("find inventory for SKU %s and location %s: %w", sku, location, err)
				}

				// Update existing inventory
				inventory.Quantity += tools.IntToFloat64(items[i].ExpectedQuantity)
				inventory.UpdatedAt = time.Now()

				if err := tx.Save(&inventory).Error; err != nil {
					return errors.New("failed to update inventory")
				}
			}

			// Create inventory movement
			mov := &database.InventoryMovement{
				SKU:            sku,
				Location:       location,
				MovementType:   "inbound",
				Quantity:       tools.IntToFloat64(items[i].ExpectedQuantity),
				RemainingStock: inventory.Quantity,
				Reason:         tools.StrPtr("receiving task completion"),
				CreatedBy:      userId,
				CreatedAt:      tools.GetCurrentTime(),
			}

			if err := tx.Create(mov).Error; err != nil {
				return fmt.Errorf("create inventory movement: %w", err)
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

					if err := tx.Where("serial_number = ? AND sku = ?", serial.SerialNumber, items[i].SKU).First(&serialItem).Error; err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							// Create new serial
							serialItem = database.Serial{
								SerialNumber: serial.SerialNumber,
								SKU:          items[i].SKU,
								CreatedAt:    tools.GetCurrentTime(),
								UpdatedAt:    tools.GetCurrentTime(),
								Status:       "available",
							}
							if err := tx.Create(&serialItem).Error; err != nil {
								return errors.New("failed to create serial")
							}
						}
					}

					inventorySerial := &database.InventorySerial{
						InventoryID: inventory.ID,
						SerialID:    serialItem.ID,
						Location:    items[i].Location,
					}

					if err := tx.Create(inventorySerial).Error; err != nil {
						return errors.New("failed to create inventory_serial association")
					}

					// Mark serial as completed
					items[i].SerialNumbers[k].Status = "completed"
					items[i].SerialNumbers[k].ID = serialItem.ID
				}

				// Mark item as completed
				items[i].Status = tools.StrPtr("completed")
			}

			if article.TrackByLot && items[i].LotNumbers != nil {
				// Check if sum of lot quantities matches expected quantity
				var totalLotQty float64
				for _, lot := range items[i].LotNumbers {
					totalLotQty += lot.Quantity
				}

				if totalLotQty != float64(items[i].ExpectedQuantity) {
					// If not, then this task can't be completed fully
					*handledResp = responses.InternalResponse{Message: fmt.Sprintf("La suma de las cantidades de lotes (%.2f) no coincide con la cantidad esperada (%d) para SKU %s", totalLotQty, items[i].ExpectedQuantity, sku), Handled: true}

					return nil
				}

				for j := 0; j < len(items[i].LotNumbers); j++ {
					lotNum := items[i].LotNumbers[j]

					var lot database.Lot

					if err := tx.Where("lot_number = ? AND sku = ?", lotNum.LotNumber, items[i].SKU).First(&lot).Error; err != nil {
						return errors.New("failed to retrieve existing lot")
					}

					// Update lot status to available
					lot.Status = tools.StrPtr("available")
					lot.UpdatedAt = tools.GetCurrentTime()

					if err := tx.Save(&lot).Error; err != nil {
						return errors.New("failed to update lot status")
					}

					inventoryLot := &database.InventoryLot{
						InventoryID: inventory.ID,
						LotID:       lot.ID,
						Quantity:    lotNum.Quantity,
						Location:    items[i].Location,
					}

					if err := tx.Create(inventoryLot).Error; err != nil {
						return errors.New("failed to create inventory_lot association")
					}

					// Set lotNum.Status to completed
					items[i].LotNumbers[j].Status = tools.StrPtr("completed")
					items[i].LotNumbers[j].ReceivedQuantity = &lotNum.Quantity
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
			"status":       "closed",
			"completed_at": tools.GetCurrentTime(),
			"updated_at":   tools.GetCurrentTime(),
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
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

func (r *ReceivingTasksRepository) CompleteReceivingLine(id int, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	handledResp := &responses.InternalResponse{}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var task database.ReceivingTask

		if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				*handledResp = responses.InternalResponse{Message: "Tarea de recepción no encontrada", Handled: true}
				return nil
			}
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al recuperar la tarea de recepción"}
			return nil
		}

		var items []requests.ReceivingTaskItemRequest
		var foundItem requests.ReceivingTaskItemRequest

		if err := json.Unmarshal(task.Items, &items); err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Formato de items inválido", Handled: true}
			return nil
		}

		found := false
		for i := 0; i < len(items); i++ {
			if items[i].SKU == item.SKU {
				found = true
				foundItem = items[i]
				break
			}
		}

		if !found {
			*handledResp = responses.InternalResponse{Message: "SKU no encontrado en los items de la tarea", Handled: true}
			return nil
		}

		var article database.Article

		if err := tx.Where("sku = ?", item.SKU).First(&article).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				*handledResp = responses.InternalResponse{Message: "Artículo no encontrado para SKU", Handled: true}
				return nil
			}
			return fmt.Errorf("find article %s: %w", item.SKU, err)
		}

		if foundItem.Status != nil && (*foundItem.Status == "completed" || *foundItem.Status == "closed" || *foundItem.Status == "partial") {
			*handledResp = responses.InternalResponse{Message: "La línea de recepción ya ha sido procesada", Handled: true}
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

		// Compare with database expected quantity
		if qty <= 0 || qty < float64(foundItem.ExpectedQuantity) {
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
				*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
				return nil
			}
		} else {
			// If given quantity meets or exceeds expected, mark item as completed
			for i := 0; i < len(items); i++ {
				if items[i].SKU == item.SKU {
					items[i].Status = tools.StrPtr("completed")
					items[i].ReceivedQuantity = tools.IntToPtr(int(qty))
					break
				}
			}
		}

		var inventory database.Inventory

		inventoryCount := int64(0)

		err := tx.Model(&database.Inventory{}).Where("sku = ? AND location = ?", item.SKU, location).Count(&inventoryCount).Error

		if err != nil {
			return fmt.Errorf("check inventory for SKU %s and location %s: %w", item.SKU, location, err)
		}

		if inventoryCount == 0 {
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
				return errors.New("failed to create inventory")
			}
		} else {
			if err := tx.Where("sku = ? AND location = ?", item.SKU, location).First(&inventory).Error; err != nil {
				return fmt.Errorf("find inventory for SKU %s and location %s: %w", item.SKU, location, err)
			}

			// Update existing inventory
			inventory.Quantity += qty
			inventory.UpdatedAt = time.Now()

			if err := tx.Save(&inventory).Error; err != nil {
				return errors.New("failed to update inventory")
			}
		}

		// Create inventory movement
		mov := &database.InventoryMovement{
			SKU:            item.SKU,
			Location:       location,
			MovementType:   "inbound",
			Quantity:       qty,
			RemainingStock: inventory.Quantity,
			Reason:         tools.StrPtr("receiving task line completion"),
			CreatedBy:      userId,
			CreatedAt:      tools.GetCurrentTime(),
		}

		if err := tx.Create(mov).Error; err != nil {
			return fmt.Errorf("create inventory movement: %w", err)
		}

		if article.TrackBySerial && item.SerialNumbers != nil {
			for k := 0; k < len(item.SerialNumbers); k++ {
				serial := item.SerialNumbers[k]

				var serialItem database.Serial

				if err := tx.Where("serial_number = ? AND sku = ?", serial.SerialNumber, item.SKU).First(&serialItem).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// Create new serial
						serialItem = database.Serial{
							SerialNumber: serial.SerialNumber,
							SKU:          item.SKU,
							CreatedAt:    tools.GetCurrentTime(),
							UpdatedAt:    tools.GetCurrentTime(),
							Status:       "available",
						}

						if err := tx.Create(&serialItem).Error; err != nil {
							return errors.New("failed to create serial")
						}

						for i := 0; i < len(items); i++ {
							if items[i].SKU == item.SKU {
								items[i].SerialNumbers = append(items[i].SerialNumbers, database.Serial{
									ID:           serialItem.ID,
									SerialNumber: serial.SerialNumber,
									SKU:          item.SKU,
									Status:       "completed",
									CreatedAt:    serialItem.CreatedAt,
									UpdatedAt:    serialItem.UpdatedAt,
								})
								break
							}
						}
					}
				}

				inventorySerial := &database.InventorySerial{
					InventoryID: inventory.ID,
					SerialID:    serialItem.ID,
					Location:    item.Location,
				}

				if err := tx.Create(inventorySerial).Error; err != nil {
					return errors.New("failed to create inventory_serial association")
				}

				// Mark serial as completed in items
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

			// If given serials count matches expected quantity, mark item as completed
			if len(item.SerialNumbers) == foundItem.ExpectedQuantity {
				item.Status = tools.StrPtr("completed")
			} else {
				item.Status = tools.StrPtr("partial")
			}
		}

		if article.TrackByLot && item.LotNumbers != nil {
			for j := 0; j < len(item.LotNumbers); j++ {
				lotNum := item.LotNumbers[j]

				var lotCount int64
				err := tx.Model(&database.Lot{}).
					Where("lot_number = ? AND sku = ?", lotNum.LotNumber, item.SKU).
					Count(&lotCount).Error
				if err != nil {
					return errors.New("failed to check existing lot")
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
						Status:         tools.StrPtr("available"),
						CreatedAt:      tools.GetCurrentTime(),
						UpdatedAt:      tools.GetCurrentTime(),
					}

					if err := tx.Create(lot).Error; err != nil {
						return errors.New("failed to create lot")
					}

					lotId = lot.ID

					// Add the new lot to items
					for i := 0; i < len(items); i++ {
						if items[i].SKU == item.SKU {
							items[i].LotNumbers = append(items[i].LotNumbers, requests.CreateLotRequest{
								LotNumber:        lot.LotNumber,
								Quantity:         lot.Quantity,
								ExpirationDate:   lotNum.ExpirationDate,
								Status:           tools.StrPtr("completed"),
								ReceivedQuantity: &lot.Quantity,
							})
							break
						}
					}

				} else {
					var lot database.Lot
					if err := tx.Where("lot_number = ? AND sku = ?", lotNum.LotNumber, item.SKU).First(&lot).Error; err != nil {
						return errors.New("failed to retrieve existing lot")
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
							*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
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
							*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
							return nil
						}

						// Update lot status to available
						lot.Status = tools.StrPtr("available")
						lot.UpdatedAt = tools.GetCurrentTime()

						if err := tx.Save(&lot).Error; err != nil {
							return errors.New("failed to update lot status")
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
					return errors.New("failed to create inventory_lot association")
				}
			}

			for i := 0; i < len(items); i++ {
				if items[i].SKU == item.SKU {
					if qty >= float64(foundItem.ExpectedQuantity) {
						items[i].Status = tools.StrPtr("completed")
					} else {
						items[i].Status = tools.StrPtr("partial")
					}
					break
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
			"items":      updatedItems,
			"updated_at": tools.GetCurrentTime(),
		}

		if err := tx.Model(&task).Updates(clean).Error; err != nil {
			*handledResp = responses.InternalResponse{Error: err, Message: "Error al actualizar la tarea de recepción"}
			return nil
		}

		return nil
	})

	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error en la transacción"}
	}

	if handledResp.Error != nil || handledResp.Handled {
		return handledResp
	}

	return nil
}
