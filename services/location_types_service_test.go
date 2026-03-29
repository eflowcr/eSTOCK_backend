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

// mockLocationTypesRepo is an in-memory fake for unit testing LocationTypesService.
type mockLocationTypesRepo struct {
	locationTypes     []database.LocationType
	listErr           *responses.InternalResponse
	listAdminResult   []database.LocationType
	listAdminErr      *responses.InternalResponse
	byID              map[string]*database.LocationType
	byIDErr           *responses.InternalResponse
	byCode            map[string]*database.LocationType
	byCodeErr         *responses.InternalResponse
	createResult      *database.LocationType
	createErr         *responses.InternalResponse
	updateResult      *database.LocationType
	updateErr         *responses.InternalResponse
	deleteErr         *responses.InternalResponse
}

func (m *mockLocationTypesRepo) ListLocationTypes() ([]database.LocationType, *responses.InternalResponse) {
	return m.locationTypes, m.listErr
}

func (m *mockLocationTypesRepo) ListLocationTypesAdmin() ([]database.LocationType, *responses.InternalResponse) {
	return m.listAdminResult, m.listAdminErr
}

func (m *mockLocationTypesRepo) GetLocationTypeByID(id string) (*database.LocationType, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if lt, ok := m.byID[id]; ok {
			return lt, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Location type not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockLocationTypesRepo) GetLocationTypeByCode(code string) (*database.LocationType, *responses.InternalResponse) {
	if m.byCodeErr != nil {
		return nil, m.byCodeErr
	}
	if m.byCode != nil {
		if lt, ok := m.byCode[code]; ok {
			return lt, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Location type not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockLocationTypesRepo) CreateLocationType(req *requests.LocationTypeCreate) (*database.LocationType, *responses.InternalResponse) {
	return m.createResult, m.createErr
}

func (m *mockLocationTypesRepo) UpdateLocationType(id string, req *requests.LocationTypeUpdate) (*database.LocationType, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}

func (m *mockLocationTypesRepo) DeleteLocationType(id string) *responses.InternalResponse {
	return m.deleteErr
}

func TestLocationTypesService_ListLocationTypes_Success(t *testing.T) {
	lts := []database.LocationType{
		{ID: "lt-1", Code: "RACK", Name: "Rack", IsActive: true},
		{ID: "lt-2", Code: "SHELF", Name: "Shelf", IsActive: true},
	}
	repo := &mockLocationTypesRepo{locationTypes: lts}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.ListLocationTypes()
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.Equal(t, "RACK", result[0].Code)
	assert.Equal(t, "Shelf", result[1].Name)
}

func TestLocationTypesService_ListLocationTypes_Error(t *testing.T) {
	repo := &mockLocationTypesRepo{
		listErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching location types",
			Handled: false,
		},
	}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.ListLocationTypes()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestLocationTypesService_ListLocationTypesAdmin_Success(t *testing.T) {
	lts := []database.LocationType{
		{ID: "lt-1", Code: "RACK", Name: "Rack", IsActive: true},
		{ID: "lt-2", Code: "SHELF", Name: "Shelf", IsActive: false},
	}
	repo := &mockLocationTypesRepo{listAdminResult: lts}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.ListLocationTypesAdmin()
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.False(t, result[1].IsActive)
}

func TestLocationTypesService_GetLocationTypeByID_Found(t *testing.T) {
	lt := &database.LocationType{ID: "lt-1", Code: "RACK", Name: "Rack"}
	repo := &mockLocationTypesRepo{
		byID: map[string]*database.LocationType{"lt-1": lt},
	}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.GetLocationTypeByID("lt-1")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "RACK", result.Code)
}

func TestLocationTypesService_GetLocationTypeByID_NotFound(t *testing.T) {
	repo := &mockLocationTypesRepo{byID: map[string]*database.LocationType{}}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.GetLocationTypeByID("lt-99")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

func TestLocationTypesService_GetLocationTypeByCode_Found(t *testing.T) {
	lt := &database.LocationType{ID: "lt-1", Code: "RACK", Name: "Rack"}
	repo := &mockLocationTypesRepo{
		byCode: map[string]*database.LocationType{"RACK": lt},
	}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.GetLocationTypeByCode("RACK")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "lt-1", result.ID)
}

func TestLocationTypesService_GetLocationTypeByCode_NotFound(t *testing.T) {
	repo := &mockLocationTypesRepo{byCode: map[string]*database.LocationType{}}
	svc := NewLocationTypesService(repo)

	result, errResp := svc.GetLocationTypeByCode("UNKNOWN")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestLocationTypesService_CreateLocationType_Success(t *testing.T) {
	isActive := true
	created := &database.LocationType{ID: "lt-new", Code: "BIN", Name: "Bin", IsActive: true}
	repo := &mockLocationTypesRepo{createResult: created}
	svc := NewLocationTypesService(repo)

	req := &requests.LocationTypeCreate{
		Code:      "BIN",
		Name:      "Bin",
		SortOrder: 3,
		IsActive:  &isActive,
	}
	result, errResp := svc.CreateLocationType(req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "BIN", result.Code)
}

func TestLocationTypesService_CreateLocationType_Conflict(t *testing.T) {
	repo := &mockLocationTypesRepo{
		createErr: &responses.InternalResponse{
			Message:    "Location type with this code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewLocationTypesService(repo)

	isActive := true
	req := &requests.LocationTypeCreate{Code: "RACK", Name: "Rack", IsActive: &isActive}
	result, errResp := svc.CreateLocationType(req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestLocationTypesService_UpdateLocationType_Success(t *testing.T) {
	isActive := true
	updated := &database.LocationType{ID: "lt-1", Code: "RACK2", Name: "Rack Updated"}
	repo := &mockLocationTypesRepo{updateResult: updated}
	svc := NewLocationTypesService(repo)

	req := &requests.LocationTypeUpdate{Code: "RACK2", Name: "Rack Updated", IsActive: &isActive}
	result, errResp := svc.UpdateLocationType("lt-1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Rack Updated", result.Name)
}

func TestLocationTypesService_UpdateLocationType_NotFound(t *testing.T) {
	repo := &mockLocationTypesRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Location type not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewLocationTypesService(repo)

	isActive := true
	req := &requests.LocationTypeUpdate{Code: "NONE", Name: "None", IsActive: &isActive}
	result, errResp := svc.UpdateLocationType("lt-99", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestLocationTypesService_DeleteLocationType_Success(t *testing.T) {
	repo := &mockLocationTypesRepo{}
	svc := NewLocationTypesService(repo)

	errResp := svc.DeleteLocationType("lt-1")
	require.Nil(t, errResp)
}

func TestLocationTypesService_DeleteLocationType_NotFound(t *testing.T) {
	repo := &mockLocationTypesRepo{
		deleteErr: &responses.InternalResponse{
			Message:    "Location type not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewLocationTypesService(repo)

	errResp := svc.DeleteLocationType("lt-99")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}
