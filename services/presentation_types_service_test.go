package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPresentationTypesRepo is an in-memory fake for unit testing PresentationTypesService.
type mockPresentationTypesRepo struct {
	types         []database.PresentationType
	listErr       *responses.InternalResponse
	adminTypes    []database.PresentationType
	adminListErr  *responses.InternalResponse
	byID          map[string]*database.PresentationType
	byIDErr       *responses.InternalResponse
	byCode        map[string]*database.PresentationType
	byCodeErr     *responses.InternalResponse
	createResult  *database.PresentationType
	createErr     *responses.InternalResponse
	updateResult  *database.PresentationType
	updateErr     *responses.InternalResponse
	deleteErr     *responses.InternalResponse
}

func (m *mockPresentationTypesRepo) ListPresentationTypes() ([]database.PresentationType, *responses.InternalResponse) {
	return m.types, m.listErr
}

func (m *mockPresentationTypesRepo) ListPresentationTypesAdmin() ([]database.PresentationType, *responses.InternalResponse) {
	return m.adminTypes, m.adminListErr
}

func (m *mockPresentationTypesRepo) GetPresentationTypeByID(id string) (*database.PresentationType, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if pt, ok := m.byID[id]; ok {
			return pt, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Presentation type not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPresentationTypesRepo) GetPresentationTypeByCode(code string) (*database.PresentationType, *responses.InternalResponse) {
	if m.byCodeErr != nil {
		return nil, m.byCodeErr
	}
	if m.byCode != nil {
		if pt, ok := m.byCode[code]; ok {
			return pt, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Presentation type not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPresentationTypesRepo) CreatePresentationType(req *requests.PresentationTypeCreate) (*database.PresentationType, *responses.InternalResponse) {
	return m.createResult, m.createErr
}

func (m *mockPresentationTypesRepo) UpdatePresentationType(id string, req *requests.PresentationTypeUpdate) (*database.PresentationType, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}

func (m *mockPresentationTypesRepo) DeletePresentationType(id string) *responses.InternalResponse {
	return m.deleteErr
}

// --- Tests ---

func TestPresentationTypesService_ListPresentationTypes_Success(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		types: []database.PresentationType{
			{ID: "1", Code: "UNIT", Name: "Unidad", IsActive: true},
			{ID: "2", Code: "BOX", Name: "Caja", IsActive: true},
		},
	}
	svc := NewPresentationTypesService(repo)
	list, errResp := svc.ListPresentationTypes()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "UNIT", list[0].Code)
	assert.Equal(t, "BOX", list[1].Code)
}

func TestPresentationTypesService_ListPresentationTypes_Error(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		listErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error listing presentation types",
			Handled: false,
		},
	}
	svc := NewPresentationTypesService(repo)
	list, errResp := svc.ListPresentationTypes()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
}

func TestPresentationTypesService_ListPresentationTypesAdmin_Success(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		adminTypes: []database.PresentationType{
			{ID: "1", Code: "UNIT", Name: "Unidad", IsActive: true},
			{ID: "2", Code: "BOX", Name: "Caja", IsActive: false},
		},
	}
	svc := NewPresentationTypesService(repo)
	list, errResp := svc.ListPresentationTypesAdmin()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.False(t, list[1].IsActive)
}

func TestPresentationTypesService_ListPresentationTypesAdmin_Error(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		adminListErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error listing admin types",
			Handled: false,
		},
	}
	svc := NewPresentationTypesService(repo)
	list, errResp := svc.ListPresentationTypesAdmin()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
}

func TestPresentationTypesService_GetPresentationTypeByID_Found(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		byID: map[string]*database.PresentationType{
			"1": {ID: "1", Code: "UNIT", Name: "Unidad", IsActive: true},
		},
	}
	svc := NewPresentationTypesService(repo)
	pt, errResp := svc.GetPresentationTypeByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, pt)
	assert.Equal(t, "UNIT", pt.Code)
}

func TestPresentationTypesService_GetPresentationTypeByID_NotFound(t *testing.T) {
	repo := &mockPresentationTypesRepo{byID: map[string]*database.PresentationType{}}
	svc := NewPresentationTypesService(repo)
	pt, errResp := svc.GetPresentationTypeByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, pt)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationTypesService_GetPresentationTypeByCode_Found(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		byCode: map[string]*database.PresentationType{
			"UNIT": {ID: "1", Code: "UNIT", Name: "Unidad", IsActive: true},
		},
	}
	svc := NewPresentationTypesService(repo)
	pt, errResp := svc.GetPresentationTypeByCode("UNIT")
	require.Nil(t, errResp)
	require.NotNil(t, pt)
	assert.Equal(t, "Unidad", pt.Name)
}

func TestPresentationTypesService_GetPresentationTypeByCode_NotFound(t *testing.T) {
	repo := &mockPresentationTypesRepo{byCode: map[string]*database.PresentationType{}}
	svc := NewPresentationTypesService(repo)
	pt, errResp := svc.GetPresentationTypeByCode("UNKNOWN")
	require.NotNil(t, errResp)
	assert.Nil(t, pt)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationTypesService_CreatePresentationType_Success(t *testing.T) {
	isActive := true
	expected := &database.PresentationType{ID: "1", Code: "PAL", Name: "Pallet", IsActive: true}
	repo := &mockPresentationTypesRepo{createResult: expected}
	svc := NewPresentationTypesService(repo)
	req := &requests.PresentationTypeCreate{
		Code:     "PAL",
		Name:     "Pallet",
		IsActive: &isActive,
	}
	result, errResp := svc.CreatePresentationType(req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "PAL", result.Code)
}

func TestPresentationTypesService_CreatePresentationType_Conflict(t *testing.T) {
	isActive := true
	repo := &mockPresentationTypesRepo{
		createErr: &responses.InternalResponse{
			Message:    "Presentation type with this code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewPresentationTypesService(repo)
	req := &requests.PresentationTypeCreate{Code: "UNIT", Name: "Unidad", IsActive: &isActive}
	result, errResp := svc.CreatePresentationType(req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestPresentationTypesService_UpdatePresentationType_Success(t *testing.T) {
	isActive := false
	expected := &database.PresentationType{ID: "1", Code: "UNIT", Name: "Unidad Updated", IsActive: false}
	repo := &mockPresentationTypesRepo{updateResult: expected}
	svc := NewPresentationTypesService(repo)
	req := &requests.PresentationTypeUpdate{
		Code:     "UNIT",
		Name:     "Unidad Updated",
		IsActive: &isActive,
	}
	result, errResp := svc.UpdatePresentationType("1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Unidad Updated", result.Name)
}

func TestPresentationTypesService_UpdatePresentationType_NotFound(t *testing.T) {
	isActive := true
	repo := &mockPresentationTypesRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Presentation type not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewPresentationTypesService(repo)
	req := &requests.PresentationTypeUpdate{Code: "UNIT", Name: "Unidad", IsActive: &isActive}
	result, errResp := svc.UpdatePresentationType("99", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationTypesService_DeletePresentationType_Success(t *testing.T) {
	repo := &mockPresentationTypesRepo{}
	svc := NewPresentationTypesService(repo)
	errResp := svc.DeletePresentationType("1")
	require.Nil(t, errResp)
}

func TestPresentationTypesService_DeletePresentationType_Error(t *testing.T) {
	repo := &mockPresentationTypesRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error deleting presentation type",
			Handled: false,
		},
	}
	svc := NewPresentationTypesService(repo)
	errResp := svc.DeletePresentationType("1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
