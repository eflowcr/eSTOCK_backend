package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// SerialsRepository is the GORM-backed implementation of ports.SerialsRepository.
//
// S3.5 W2-A: every method is tenant-scoped via WHERE tenant_id = ?.
type SerialsRepository struct {
	DB *gorm.DB
}

func (r *SerialsRepository) GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse) {
	var serial database.Serial

	err := r.DB.Table(database.Serial{}.TableName()).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&serial).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Message:    "Serie no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la serie",
			Handled: false,
		}
	}

	return &serial, nil
}

func (r *SerialsRepository) GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse) {
	var serials []database.Serial

	err := r.DB.Table(database.Serial{}.TableName()).
		Where("sku = ? AND tenant_id = ?", sku, tenantID).
		Order("created_at DESC").
		Find(&serials).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener las series",
			Handled: false,
		}
	}

	return serials, nil
}

func (r *SerialsRepository) CreateSerial(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse {
	now := tools.GetCurrentTime()

	serial := &database.Serial{
		TenantID:     tenantID,
		SerialNumber: data.SerialNumber,
		SKU:          data.SKU,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := r.DB.Create(serial).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear la serie",
			Handled: false,
		}
	}

	return nil
}

func (r *SerialsRepository) UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	var serial database.Serial

	err := r.DB.First(&serial, "id = ? AND tenant_id = ?", id, tenantID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Message:    "Serie no encontrada",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la serie",
			Handled: false,
		}
	}

	protectedFields := map[string]bool{
		"id":         true,
		"created_at": true,
		"tenant_id":  true, // S3.5 W2-A: tenant_id is immutable after creation.
	}

	for k := range protectedFields {
		delete(data, k)
	}

	data["updated_at"] = tools.GetCurrentTime()

	if err := r.DB.Model(&serial).Updates(data).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar la serie",
			Handled: false,
		}
	}

	return nil
}

func (r *SerialsRepository) DeleteSerial(tenantID, id string) *responses.InternalResponse {
	result := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&database.Serial{})
	if result.Error != nil {
		return &responses.InternalResponse{
			Error:   result.Error,
			Message: "Error al eliminar la serie",
			Handled: false,
		}
	}

	if result.RowsAffected == 0 {
		return &responses.InternalResponse{
			Message:    "Serie no encontrada",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}

	return nil
}
