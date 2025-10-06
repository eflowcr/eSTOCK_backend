package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type PresentationsRepository struct {
	DB *gorm.DB
}

func (r *PresentationsRepository) GetAllPresentations() ([]database.Presentations, *responses.InternalResponse) {
	var presentations []database.Presentations

	err := r.DB.
		Table(database.Presentations{}.TableName()).
		Order("description ASC").
		Find(&presentations).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener las presentaciones",
			Handled: false,
		}
	}

	return presentations, nil
}

func (r *PresentationsRepository) GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse) {
	var presentation database.Presentations

	err := r.DB.
		Table(database.Presentations{}.TableName()).
		Where("presentation_id = ?", id).
		First(&presentation).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Presentación no encontrada",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la presentación",
			Handled: false,
		}
	}

	return &presentation, nil
}

func (r *PresentationsRepository) CreatePresentation(data *database.Presentations) *responses.InternalResponse {
	// Check if presentation with same id already exists

	var existingPresentation database.Presentations
	err := r.DB.
		Table(database.Presentations{}.TableName()).
		Where("presentation_id = ?", data.PresentationId).
		First(&existingPresentation).Error

	if err == nil {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Ya existe una presentación con el ID proporcionado",
			Handled: true,
		}
	} else if err != gorm.ErrRecordNotFound {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al verificar la existencia de la presentación",
			Handled: false,
		}
	}

	err = r.DB.Create(data).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear la presentación",
			Handled: false,
		}
	}

	return nil
}

func (r *PresentationsRepository) UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse) {
	var existingPresentation database.Presentations
	err := r.DB.
		Table(database.Presentations{}.TableName()).
		Where("presentation_id = ?", id).
		First(&existingPresentation).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Presentación no encontrada",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener la presentación",
			Handled: false,
		}
	}

	existingPresentation.Description = name

	err = r.DB.
		Model(&database.Presentations{}).
		Where("presentation_id = ?", id).
		Update("description", name).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar la presentación",
			Handled: false,
		}
	}

	return &existingPresentation, nil
}

func (r *PresentationsRepository) DeletePresentation(id string) *responses.InternalResponse {
	err := r.DB.
		Table(database.Presentations{}.TableName()).Where("presentation_id = ?", id).
		Delete(&database.Presentations{}).Error

	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar la presentación",
			Handled: false,
		}
	}

	return nil
}
