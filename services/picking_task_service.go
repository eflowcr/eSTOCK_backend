package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type PickingTaskService struct {
	Repository *repositories.PickingTaskRepository
}

func NewPickingTaskService(repo *repositories.PickingTaskRepository) *PickingTaskService {
	return &PickingTaskService{
		Repository: repo,
	}
}

func (s *PickingTaskService) GetAllPickingTasks() ([]database.PickingTask, *responses.InternalResponse) {
	return s.Repository.GetAllPickingTasks()
}

func (s *PickingTaskService) GetPickingTaskByID(id int) (*database.PickingTask, *responses.InternalResponse) {
	return s.Repository.GetPickingTaskByID(id)
}

func (s *PickingTaskService) CreatePickingTask(userId string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	return s.Repository.CreatePickingTask(userId, task)
}

func (s *PickingTaskService) UpdatePickingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdatePickingTask(id, data)
}

func (s *PickingTaskService) ImportPickingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return s.Repository.ImportPickingTaskFromExcel(userID, fileBytes)
}

func (s *PickingTaskService) ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportPickingTasksToExcel()
}

func (s *PickingTaskService) CompletePickingTask(id int, location, userId string) *responses.InternalResponse {
	return s.Repository.CompletePickingTask(id, location, userId)
}

func (s *PickingTaskService) CompletePickingLine(id int, location, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	return s.Repository.CompletePickingLine(id, location, userId, item)
}
