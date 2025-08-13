package repositories

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
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

	return &responses.InternalResponse{
		Message: fmt.Sprintf("Receiving task created: %s", taskID),
		Handled: true,
	}
}
