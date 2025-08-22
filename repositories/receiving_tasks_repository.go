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

	if len(tasks) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No receiving tasks found",
			Handled: true,
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
	var items []database.ReceivingTaskItem
	if err := json.Unmarshal(task.Items, &items); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
	}

	nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
	taskID := fmt.Sprintf("RCV-%06d", nowMillis%1_000_000)

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to marshal items", Handled: true}
	}

	priority := task.Priority
	if priority == "" {
		priority = "normal"
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

	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&receivingTask).Error; err != nil {
			return fmt.Errorf("create task: %w", err)
		}

		for i := 0; i < len(items); i++ {
			sku := items[i].SKU

			var invCount int64
			if err := tx.Model(&database.Inventory{}).
				Where("sku = ?", sku).
				Limit(1).
				Count(&invCount).Error; err != nil {
				return fmt.Errorf("check inventory %s: %w", sku, err)
			}
			if invCount == 0 {
				continue
			}

			var article database.Article
			if err := tx.Where("sku = ?", sku).First(&article).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				return fmt.Errorf("find article %s: %w", sku, err)
			}

			if article.TrackByLot && len(items[i].LotNumbers) > 0 {
				for j := 0; j < len(items[i].LotNumbers); j++ {
					lot := database.Lot{
						LotNumber: items[i].LotNumbers[j],
						SKU:       sku,
						CreatedAt: time.Now(),
					}

					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&lot).Error; err != nil {
						return fmt.Errorf("create lot %s/%s: %w", sku, lot.LotNumber, err)
					}
				}
			}

			if article.TrackBySerial && len(items[i].SerialNumbers) > 0 {
				for j := 0; j < len(items[i].SerialNumbers); j++ {
					serial := database.Serial{
						SerialNumber: items[i].SerialNumbers[j],
						SKU:          sku,
						CreatedAt:    time.Now(),
					}

					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&serial).Error; err != nil {
						return fmt.Errorf("create serial %s: %w", serial.SerialNumber, err)
					}
				}
			}
		}

		return nil
	}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to create receiving task", Handled: true}
	}

	return nil
}

func (r *ReceivingTasksRepository) UpdateReceivingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	var task database.ReceivingTask
	if err := r.DB.First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{Message: "Receiving task not found", Handled: true}
		}
		return &responses.InternalResponse{Error: err, Message: "Failed to retrieve receiving task"}
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
				return &responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
			}
			clean["items"] = b
		}
	}

	if err := r.DB.Model(&task).Updates(clean).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to update receiving task"}
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
