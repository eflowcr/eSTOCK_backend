package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPresentationsRepo is an in-memory fake for unit testing PresentationsService.
type mockPresentationsRepo struct {
	presentations []database.Presentations
	listErr       *responses.InternalResponse
	byID          map[string]*database.Presentations
	byIDErr       *responses.InternalResponse
	createErr     *responses.InternalResponse
	updateResult  *database.Presentations
	updateErr     *responses.InternalResponse
	deleteErr     *responses.InternalResponse
}

func (m *mockPresentationsRepo) GetAllPresentations() ([]database.Presentations, *responses.InternalResponse) {
	return m.presentations, m.listErr
}

func (m *mockPresentationsRepo) GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if p, ok := m.byID[id]; ok {
			return p, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Presentation not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPresentationsRepo) CreatePresentation(data *database.Presentations) *responses.InternalResponse {
	return m.createErr
}

func (m *mockPresentationsRepo) UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}

func (m *mockPresentationsRepo) DeletePresentation(id string) *responses.InternalResponse {
	return m.deleteErr
}

func TestPresentationsService_GetAllPresentations_Success(t *testing.T) {
	presentations := []database.Presentations{
		{PresentationId: "p-1", Description: "Unit"},
		{PresentationId: "p-2", Description: "Box"},
		{PresentationId: "p-3", Description: "Pallet"},
	}
	repo := &mockPresentationsRepo{presentations: presentations}
	svc := NewPresentationsService(repo)

	result, errResp := svc.GetAllPresentations()
	require.Nil(t, errResp)
	require.Len(t, result, 3)
	assert.Equal(t, "Unit", result[0].Description)
	assert.Equal(t, "Pallet", result[2].Description)
}

func TestPresentationsService_GetAllPresentations_Empty(t *testing.T) {
	repo := &mockPresentationsRepo{presentations: []database.Presentations{}}
	svc := NewPresentationsService(repo)

	result, errResp := svc.GetAllPresentations()
	require.Nil(t, errResp)
	assert.Empty(t, result)
}

func TestPresentationsService_GetAllPresentations_Error(t *testing.T) {
	repo := &mockPresentationsRepo{
		listErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching presentations",
			Handled: false,
		},
	}
	svc := NewPresentationsService(repo)

	result, errResp := svc.GetAllPresentations()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestPresentationsService_GetPresentationByID_Found(t *testing.T) {
	p := &database.Presentations{PresentationId: "p-1", Description: "Unit"}
	repo := &mockPresentationsRepo{
		byID: map[string]*database.Presentations{"p-1": p},
	}
	svc := NewPresentationsService(repo)

	result, errResp := svc.GetPresentationByID("p-1")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Unit", result.Description)
}

func TestPresentationsService_GetPresentationByID_NotFound(t *testing.T) {
	repo := &mockPresentationsRepo{byID: map[string]*database.Presentations{}}
	svc := NewPresentationsService(repo)

	result, errResp := svc.GetPresentationByID("p-99")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

func TestPresentationsService_CreatePresentation_Success(t *testing.T) {
	repo := &mockPresentationsRepo{}
	svc := NewPresentationsService(repo)

	data := &database.Presentations{PresentationId: "p-new", Description: "Case"}
	errResp := svc.CreatePresentation(data)
	require.Nil(t, errResp)
}

func TestPresentationsService_CreatePresentation_Conflict(t *testing.T) {
	repo := &mockPresentationsRepo{
		createErr: &responses.InternalResponse{
			Message:    "Presentation already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewPresentationsService(repo)

	data := &database.Presentations{PresentationId: "p-1", Description: "Unit"}
	errResp := svc.CreatePresentation(data)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

func TestPresentationsService_UpdatePresentation_Success(t *testing.T) {
	updated := &database.Presentations{PresentationId: "p-1", Description: "Updated Unit"}
	repo := &mockPresentationsRepo{updateResult: updated}
	svc := NewPresentationsService(repo)

	result, errResp := svc.UpdatePresentation("p-1", "Updated Unit")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Unit", result.Description)
}

func TestPresentationsService_UpdatePresentation_NotFound(t *testing.T) {
	repo := &mockPresentationsRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Presentation not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewPresentationsService(repo)

	result, errResp := svc.UpdatePresentation("p-99", "New Name")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationsService_DeletePresentation_Success(t *testing.T) {
	repo := &mockPresentationsRepo{}
	svc := NewPresentationsService(repo)

	errResp := svc.DeletePresentation("p-1")
	require.Nil(t, errResp)
}

func TestPresentationsService_DeletePresentation_Error(t *testing.T) {
	repo := &mockPresentationsRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error deleting presentation",
			Handled: false,
		},
	}
	svc := NewPresentationsService(repo)

	errResp := svc.DeletePresentation("p-1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
