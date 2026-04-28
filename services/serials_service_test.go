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
//
// S3.5 W2-A: every method accepts and records tenantID so isolation tests can
// assert the controller/service forwarded the right tenant.
type mockSerialsRepo struct {
	byID         map[string]*database.Serial
	bySKU        map[string][]database.Serial
	createErr    *responses.InternalResponse
	updateErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
	gotTenantIDs []string
}

func (m *mockSerialsRepo) recordTenant(t string) { m.gotTenantIDs = append(m.gotTenantIDs, t) }

func (m *mockSerialsRepo) GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	if m.byID != nil {
		if s, ok := m.byID[id]; ok {
			if s.TenantID == "" || s.TenantID == tenantID {
				return s, nil
			}
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "serial no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockSerialsRepo) GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	if m.bySKU != nil {
		if serials, ok := m.bySKU[sku]; ok {
			out := make([]database.Serial, 0, len(serials))
			for _, s := range serials {
				if s.TenantID == "" || s.TenantID == tenantID {
					out = append(out, s)
				}
			}
			return out, nil
		}
	}
	return []database.Serial{}, nil
}

func (m *mockSerialsRepo) CreateSerial(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse {
	m.recordTenant(tenantID)
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockSerialsRepo) UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	m.recordTenant(tenantID)
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockSerialsRepo) DeleteSerial(tenantID, id string) *responses.InternalResponse {
	m.recordTenant(tenantID)
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

// --- Tests ---

func TestSerialsService_GetSerialByID_Found(t *testing.T) {
	repo := &mockSerialsRepo{
		byID: map[string]*database.Serial{
			"s1": {ID: "s1", TenantID: testTenantA, SerialNumber: "SN-001", SKU: "SKU-A", Status: "available"},
		},
	}
	svc := NewSerialsService(repo)
	serial, errResp := svc.GetSerialByID(testTenantA, "s1")
	require.Nil(t, errResp)
	require.NotNil(t, serial)
	assert.Equal(t, "SN-001", serial.SerialNumber)
	assert.Equal(t, "SKU-A", serial.SKU)
}

func TestSerialsService_GetSerialByID_NotFound(t *testing.T) {
	repo := &mockSerialsRepo{byID: map[string]*database.Serial{}}
	svc := NewSerialsService(repo)
	serial, errResp := svc.GetSerialByID(testTenantA, "missing")
	require.NotNil(t, errResp)
	assert.Nil(t, serial)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestSerialsService_GetSerialByID_TenantIsolation_blocksOtherTenant(t *testing.T) {
	repo := &mockSerialsRepo{
		byID: map[string]*database.Serial{
			"s1": {ID: "s1", TenantID: testTenantA, SerialNumber: "SN-001", SKU: "SKU-A", Status: "available"},
		},
	}
	svc := NewSerialsService(repo)

	// Owner tenant sees it.
	got, err := svc.GetSerialByID(testTenantA, "s1")
	require.Nil(t, err)
	require.NotNil(t, got)

	// Other tenant cannot.
	got2, err2 := svc.GetSerialByID(testTenantB, "s1")
	assert.Nil(t, got2)
	require.NotNil(t, err2)
	assert.Equal(t, responses.StatusNotFound, err2.StatusCode)
}

func TestSerialsService_GetSerialsBySKU_Found(t *testing.T) {
	repo := &mockSerialsRepo{
		bySKU: map[string][]database.Serial{
			"SKU-A": {
				{ID: "s1", TenantID: testTenantA, SerialNumber: "SN-001", SKU: "SKU-A"},
				{ID: "s2", TenantID: testTenantA, SerialNumber: "SN-002", SKU: "SKU-A"},
			},
		},
	}
	svc := NewSerialsService(repo)
	serials, errResp := svc.GetSerialsBySKU(testTenantA, "SKU-A")
	require.Nil(t, errResp)
	require.Len(t, serials, 2)
	assert.Equal(t, "SN-001", serials[0].SerialNumber)
}

func TestSerialsService_GetSerialsBySKU_NotFound(t *testing.T) {
	repo := &mockSerialsRepo{bySKU: map[string][]database.Serial{}}
	svc := NewSerialsService(repo)
	serials, errResp := svc.GetSerialsBySKU(testTenantA, "UNKNOWN-SKU")
	require.Nil(t, errResp)
	assert.Empty(t, serials)
}

func TestSerialsService_GetSerialsBySKU_TenantIsolation_returnsOnlyOwnTenant(t *testing.T) {
	repo := &mockSerialsRepo{
		bySKU: map[string][]database.Serial{
			"SKU-X": {
				{ID: "a1", TenantID: testTenantA, SerialNumber: "SN-A1", SKU: "SKU-X"},
				{ID: "b1", TenantID: testTenantB, SerialNumber: "SN-B1", SKU: "SKU-X"},
			},
		},
	}
	svc := NewSerialsService(repo)

	listA, err := svc.GetSerialsBySKU(testTenantA, "SKU-X")
	require.Nil(t, err)
	require.Len(t, listA, 1)
	assert.Equal(t, "a1", listA[0].ID)

	listB, err := svc.GetSerialsBySKU(testTenantB, "SKU-X")
	require.Nil(t, err)
	require.Len(t, listB, 1)
	assert.Equal(t, "b1", listB[0].ID)
}

func TestSerialsService_Create_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	req := &requests.CreateSerialRequest{SerialNumber: "SN-NEW", SKU: "SKU-B"}
	errResp := svc.Create(testTenantA, req)
	require.Nil(t, errResp)
	require.Contains(t, repo.gotTenantIDs, testTenantA)
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
	errResp := svc.Create(testTenantA, req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestSerialsService_UpdateSerial_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	errResp := svc.UpdateSerial(testTenantA, "s1", map[string]interface{}{"status": "used"})
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
	errResp := svc.UpdateSerial(testTenantA, "missing", map[string]interface{}{"status": "used"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestSerialsService_Delete_Success(t *testing.T) {
	repo := &mockSerialsRepo{}
	svc := NewSerialsService(repo)
	errResp := svc.Delete(testTenantA, "s1")
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
	errResp := svc.Delete(testTenantA, "s1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
