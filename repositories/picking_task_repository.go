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

	if len(tasks) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No picking tasks found",
			Handled: true,
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
	var items []database.PickingTaskItem

	if err := json.Unmarshal(task.Items, &items); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Invalid items format", Handled: true}
	}

	nowMillis := time.Now().UnixNano() / int64(time.Millisecond)
	taskID := fmt.Sprintf("PICK-%06d", nowMillis%1_000_000)

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to marshal items", Handled: true}
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

	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pickingTask).Error; err != nil {
			return fmt.Errorf("create picking task: %w", err)
		}
		return nil
	}); err != nil {
		return &responses.InternalResponse{Error: err, Message: "Failed to create picking task", Handled: true}
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
