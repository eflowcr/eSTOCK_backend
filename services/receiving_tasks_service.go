package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type ReceivingTasksService struct {
	Repository *repositories.ReceivingTasksRepository
}

func NewReceivingTasksService(repo *repositories.ReceivingTasksRepository) *ReceivingTasksService {
	return &ReceivingTasksService{
		Repository: repo,
	}
}

func (s *ReceivingTasksService) GetAllReceivingTasks() ([]database.ReceivingTask, *responses.InternalResponse) {
	return s.Repository.GetAllReceivingTasks()
}

func (s *ReceivingTasksService) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	return s.Repository.CreateReceivingTask(userId, task)
}

func (s *ReceivingTasksService) UpdateReceivingTask(id int, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateReceivingTask(id, data)
}
