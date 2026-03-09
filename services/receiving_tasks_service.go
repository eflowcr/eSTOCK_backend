package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type ReceivingTasksService struct {
	Repository ports.ReceivingTasksRepository
}

func NewReceivingTasksService(repo ports.ReceivingTasksRepository) *ReceivingTasksService {
	return &ReceivingTasksService{
		Repository: repo,
	}
}

func (s *ReceivingTasksService) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	return s.Repository.GetAllReceivingTasks()
}

func (s *ReceivingTasksService) GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse) {
	return s.Repository.GetReceivingTaskByID(id)
}

func (s *ReceivingTasksService) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	return s.Repository.CreateReceivingTask(userId, task)
}

func (s *ReceivingTasksService) UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateReceivingTask(id, data)
}

func (s *ReceivingTasksService) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return s.Repository.ImportReceivingTaskFromExcel(userID, fileBytes)
}

func (s *ReceivingTasksService) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportReceivingTaskToExcel()
}

func (s *ReceivingTasksService) CompleteFullTask(id string, location, userId string) *responses.InternalResponse {
	return s.Repository.CompleteFullTask(id, location, userId)
}

func (s *ReceivingTasksService) CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	return s.Repository.CompleteReceivingLine(id, location, userId, item)
}
