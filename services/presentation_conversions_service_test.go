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

// mockPresentationConversionsRepo is an in-memory fake for unit testing PresentationConversionsService.
type mockPresentationConversionsRepo struct {
	conversions      []database.PresentationConversion
	listErr          *responses.InternalResponse
	adminConversions []database.PresentationConversion
	adminListErr     *responses.InternalResponse
	byID             map[string]*database.PresentationConversion
	byIDErr          *responses.InternalResponse
	byFromTo         map[string]*database.PresentationConversion
	byFromToErr      *responses.InternalResponse
	createResult     *database.PresentationConversion
	createErr        *responses.InternalResponse
	updateResult     *database.PresentationConversion
	updateErr        *responses.InternalResponse
	deleteErr        *responses.InternalResponse
}

func (m *mockPresentationConversionsRepo) ListPresentationConversions() ([]database.PresentationConversion, *responses.InternalResponse) {
	return m.conversions, m.listErr
}

func (m *mockPresentationConversionsRepo) ListPresentationConversionsAdmin() ([]database.PresentationConversion, *responses.InternalResponse) {
	return m.adminConversions, m.adminListErr
}

func (m *mockPresentationConversionsRepo) GetPresentationConversionByID(id string) (*database.PresentationConversion, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if pc, ok := m.byID[id]; ok {
			return pc, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Presentation conversion not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPresentationConversionsRepo) GetPresentationConversionByFromAndTo(fromID, toID string) (*database.PresentationConversion, *responses.InternalResponse) {
	if m.byFromToErr != nil {
		return nil, m.byFromToErr
	}
	key := fromID + ":" + toID
	if m.byFromTo != nil {
		if pc, ok := m.byFromTo[key]; ok {
			return pc, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Presentation conversion not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPresentationConversionsRepo) CreatePresentationConversion(req *requests.PresentationConversionCreate) (*database.PresentationConversion, *responses.InternalResponse) {
	return m.createResult, m.createErr
}

func (m *mockPresentationConversionsRepo) UpdatePresentationConversion(id string, req *requests.PresentationConversionUpdate) (*database.PresentationConversion, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}

func (m *mockPresentationConversionsRepo) DeletePresentationConversion(id string) *responses.InternalResponse {
	return m.deleteErr
}

// --- Tests ---

func TestPresentationConversionsService_ListPresentationConversions_Success(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		conversions: []database.PresentationConversion{
			{ID: "1", FromPresentationTypeID: "pt-1", ToPresentationTypeID: "pt-2", ConversionFactor: 12.0, IsActive: true},
			{ID: "2", FromPresentationTypeID: "pt-2", ToPresentationTypeID: "pt-3", ConversionFactor: 10.0, IsActive: true},
		},
	}
	svc := NewPresentationConversionsService(repo)
	list, errResp := svc.ListPresentationConversions()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, float64(12), list[0].ConversionFactor)
}

func TestPresentationConversionsService_ListPresentationConversions_Error(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		listErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error listing conversions",
			Handled: false,
		},
	}
	svc := NewPresentationConversionsService(repo)
	list, errResp := svc.ListPresentationConversions()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
}

func TestPresentationConversionsService_ListPresentationConversionsAdmin_Success(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		adminConversions: []database.PresentationConversion{
			{ID: "1", FromPresentationTypeID: "pt-1", ToPresentationTypeID: "pt-2", ConversionFactor: 12.0, IsActive: true},
			{ID: "2", FromPresentationTypeID: "pt-2", ToPresentationTypeID: "pt-3", ConversionFactor: 10.0, IsActive: false},
		},
	}
	svc := NewPresentationConversionsService(repo)
	list, errResp := svc.ListPresentationConversionsAdmin()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.False(t, list[1].IsActive)
}

func TestPresentationConversionsService_ListPresentationConversionsAdmin_Error(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		adminListErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error listing admin conversions",
			Handled: false,
		},
	}
	svc := NewPresentationConversionsService(repo)
	list, errResp := svc.ListPresentationConversionsAdmin()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
}

func TestPresentationConversionsService_GetPresentationConversionByID_Found(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		byID: map[string]*database.PresentationConversion{
			"1": {ID: "1", FromPresentationTypeID: "pt-1", ToPresentationTypeID: "pt-2", ConversionFactor: 12.0, IsActive: true},
		},
	}
	svc := NewPresentationConversionsService(repo)
	pc, errResp := svc.GetPresentationConversionByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, pc)
	assert.Equal(t, float64(12), pc.ConversionFactor)
}

func TestPresentationConversionsService_GetPresentationConversionByID_NotFound(t *testing.T) {
	repo := &mockPresentationConversionsRepo{byID: map[string]*database.PresentationConversion{}}
	svc := NewPresentationConversionsService(repo)
	pc, errResp := svc.GetPresentationConversionByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, pc)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationConversionsService_GetPresentationConversionByFromAndTo_Found(t *testing.T) {
	key := "pt-1:pt-2"
	repo := &mockPresentationConversionsRepo{
		byFromTo: map[string]*database.PresentationConversion{
			key: {ID: "1", FromPresentationTypeID: "pt-1", ToPresentationTypeID: "pt-2", ConversionFactor: 12.0, IsActive: true},
		},
	}
	svc := NewPresentationConversionsService(repo)
	pc, errResp := svc.GetPresentationConversionByFromAndTo("pt-1", "pt-2")
	require.Nil(t, errResp)
	require.NotNil(t, pc)
	assert.Equal(t, "pt-1", pc.FromPresentationTypeID)
}

func TestPresentationConversionsService_GetPresentationConversionByFromAndTo_NotFound(t *testing.T) {
	repo := &mockPresentationConversionsRepo{byFromTo: map[string]*database.PresentationConversion{}}
	svc := NewPresentationConversionsService(repo)
	pc, errResp := svc.GetPresentationConversionByFromAndTo("pt-x", "pt-y")
	require.NotNil(t, errResp)
	assert.Nil(t, pc)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationConversionsService_CreatePresentationConversion_Success(t *testing.T) {
	isActive := true
	expected := &database.PresentationConversion{
		ID:                     "1",
		FromPresentationTypeID: "pt-1",
		ToPresentationTypeID:   "pt-2",
		ConversionFactor:       12.0,
		IsActive:               true,
	}
	repo := &mockPresentationConversionsRepo{createResult: expected}
	svc := NewPresentationConversionsService(repo)
	req := &requests.PresentationConversionCreate{
		FromPresentationTypeID: "pt-1",
		ToPresentationTypeID:   "pt-2",
		ConversionFactor:       12.0,
		IsActive:               &isActive,
	}
	result, errResp := svc.CreatePresentationConversion(req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, float64(12), result.ConversionFactor)
}

func TestPresentationConversionsService_CreatePresentationConversion_Conflict(t *testing.T) {
	isActive := true
	repo := &mockPresentationConversionsRepo{
		createErr: &responses.InternalResponse{
			Message:    "Conversion rule already exists for this pair",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewPresentationConversionsService(repo)
	req := &requests.PresentationConversionCreate{
		FromPresentationTypeID: "pt-1",
		ToPresentationTypeID:   "pt-2",
		ConversionFactor:       12.0,
		IsActive:               &isActive,
	}
	result, errResp := svc.CreatePresentationConversion(req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestPresentationConversionsService_UpdatePresentationConversion_Success(t *testing.T) {
	isActive := false
	expected := &database.PresentationConversion{ID: "1", ConversionFactor: 24.0, IsActive: false}
	repo := &mockPresentationConversionsRepo{updateResult: expected}
	svc := NewPresentationConversionsService(repo)
	req := &requests.PresentationConversionUpdate{
		ConversionFactor: 24.0,
		IsActive:         &isActive,
	}
	result, errResp := svc.UpdatePresentationConversion("1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, float64(24), result.ConversionFactor)
}

func TestPresentationConversionsService_UpdatePresentationConversion_NotFound(t *testing.T) {
	isActive := true
	repo := &mockPresentationConversionsRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Presentation conversion not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewPresentationConversionsService(repo)
	req := &requests.PresentationConversionUpdate{ConversionFactor: 5.0, IsActive: &isActive}
	result, errResp := svc.UpdatePresentationConversion("99", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPresentationConversionsService_DeletePresentationConversion_Success(t *testing.T) {
	repo := &mockPresentationConversionsRepo{}
	svc := NewPresentationConversionsService(repo)
	errResp := svc.DeletePresentationConversion("1")
	require.Nil(t, errResp)
}

func TestPresentationConversionsService_DeletePresentationConversion_Error(t *testing.T) {
	repo := &mockPresentationConversionsRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error deleting conversion",
			Handled: false,
		},
	}
	svc := NewPresentationConversionsService(repo)
	errResp := svc.DeletePresentationConversion("1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
