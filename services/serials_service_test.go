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

// mockSerialsRepo is an in-memory fake for unit testing SerialsService.
type mockSerialsRepo struct {
	byID      map[string]*database.Serial
	bySKU     map[string][]database.Serial
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockSerialsRepo) GetSerialByID(id string) (*database.Serial, *responses.InternalResponse) {
	if m.byID != nil {
		if s, ok := m.byID[id]; ok {
			return s, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "serial no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockSerialsRepo) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	if m.bySKU != nil {
		if serials, ok := m.bySKU[sku]; ok {
			return serials, nil
		}
	}
	return []database.Serial{}, nil
}

func (m *mockSerialsRepo) CreateSerial(data *requests.CreateSerialRequest) *responses.InternalResponse {
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockSerialsRepo) UpdateSerial(id string, data map[string]interface{}) *responses.InternalResponse {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockSerialsRepo) DeleteSerial(id string) *responses.InternalResponse {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

// --- Tests ---

func TestSerialsService_GetSerialByID_Found(t *testing.T) {
	repo := &mockSerialsRepo{
		byID: map[string]*database.Serial{
			"s1": {ID: "s1", SerialNumber: "SN-001", SKU: "SKU-A", Status: "available"},
		},
	}
	svc := NewSerialsService(repo)
	serial, errResp := svc.GetSerialByID("s1")
	require.Nil(t, errResp)
	require.NotNil(t, serial)
	assert.Equal(t, "SN-001", serial.SerialNumber)
	assert.Equal(t, "SKU-A", serial.SKU)
}

func TestSerialsService_GetSerialByID_NotFound(t *testing.T) {
	repo := &mockSerialsRepo{byID: map[string]*database.Serial{}}
	svc := NewSerialsService(repo)
	serial, errResp := svc.GetSerialByID("missing")
	require.NotNil(t, errResp)
	assert.Nil(t, serial)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestSerialsService_GetSerialsBySKU_Found(t *testing.T) {
	repo := &mockSerialsRepo{
		bySKU: map[string][]database.Serial{
			"SKU-A": {
				{ID: "s1", SerialNumber: "SN-001", SKU: "SKU-A"},
				{ID: "s2", SerialNumber: "SN-002", SKU: "SKU-A"},
			},
		},
	}
	svc := NewSerialsService(repo)
	serials, errResp := svc.GetSerialsBySKU("SKU-A")
	require.Nil(t, errResp)
	require.Len(t, serials, 2)
	assert.Equal(t, "SN-001", serials[0].SerialNumber)
}

func TestSerialsService_GetSerialsBySKU_NotFound(t *testing.T) {
	repo := &mockSerialsRepo{bySKU: map[string][]database.Serial{}}
	svc := NewSerialsService(repo)
	serials, errResp := svc.GetSerialsBySKU("UNKNOWN-SKU")
	require.Nil(t, errResp)
	assert.Empty(t, serials)
}

func TestSerialsService_Create_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	req := &requests.CreateSerialRequest{SerialNumber: "SN-NEW", SKU: "SKU-B"}
	errResp := svc.Create(req)
	require.Nil(t, errResp)
}

func TestSerialsService_Create_Conflict(t *testing.T) {
	repo := &mockSerialsRepo{
		createErr: &responses.InternalResponse{
			Message:    "ya existe un serial con ese número",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewSerialsService(repo)
	req := &requests.CreateSerialRequest{SerialNumber: "SN-DUP", SKU: "SKU-B"}
	errResp := svc.Create(req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestSerialsService_UpdateSerial_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	errResp := svc.UpdateSerial("s1", map[string]interface{}{"status": "used"})
	require.Nil(t, errResp)
}

func TestSerialsService_UpdateSerial_NotFound(t *testing.T) {
	repo := &mockSerialsRepo{
		updateErr: &responses.InternalResponse{
			Message:    "serial no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewSerialsService(repo)
	errResp := svc.UpdateSerial("missing", map[string]interface{}{"status": "used"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestSerialsService_Delete_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	errResp := svc.Delete("s1")
	require.Nil(t, errResp)
}

func TestSerialsService_Delete_Error(t *testing.T) {
	repo := &mockSerialsRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "error al eliminar serial",
			Handled: false,
		},
	}
	svc := NewSerialsService(repo)
	errResp := svc.Delete("s1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
