package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────
//
// S3.5 W2-A: mock records the tenantID it was called with so isolation tests
// can assert the controller forwarded the right tenant.

type mockSerialsRepoCtrl struct {
	byID         map[string]*database.Serial
	bySKU        map[string][]database.Serial
	createErr    *responses.InternalResponse
	updateErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
	gotTenantIDs []string
}

func (m *mockSerialsRepoCtrl) recordTenant(t string) { m.gotTenantIDs = append(m.gotTenantIDs, t) }

func (m *mockSerialsRepoCtrl) GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	if m.byID != nil {
		if s, ok := m.byID[id]; ok {
			return s, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockSerialsRepoCtrl) GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	if m.bySKU != nil {
		if list, ok := m.bySKU[sku]; ok {
			return list, nil
		}
	}
	return []database.Serial{}, nil
}

func (m *mockSerialsRepoCtrl) CreateSerial(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return m.createErr
}

func (m *mockSerialsRepoCtrl) UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return m.updateErr
}

func (m *mockSerialsRepoCtrl) DeleteSerial(tenantID, id string) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return m.deleteErr
}

func newSerialsController(repo *mockSerialsRepoCtrl) *SerialsController {
	svc := services.NewSerialsService(repo)
	return NewSerialsController(*svc, ctrlTenantA)
}

func TestSerialsController_GetSerialByID_Success(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		byID: map[string]*database.Serial{
			"s1": {ID: "s1", SerialNumber: "SN-001", SKU: "SKU-A"},
		},
	}
	ctrl := newSerialsController(repo)
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/s1", nil, gin.Params{{Key: "id", Value: "s1"}})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, repo.gotTenantIDs, ctrlTenantA)
}

func TestSerialsController_GetSerialByID_NotFound(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{byID: map[string]*database.Serial{}})
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/missing", nil, gin.Params{{Key: "id", Value: "missing"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSerialsController_GetSerialByID_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_GetSerialsBySKU_Success(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		bySKU: map[string][]database.Serial{
			"SKU-A": {{ID: "s1", SerialNumber: "SN-001", SKU: "SKU-A"}},
		},
	}
	ctrl := newSerialsController(repo)
	w := performRequest(ctrl.GetSerialsBySKU, "GET", "/serials/by-sku/SKU-A", nil, gin.Params{{Key: "sku", Value: "SKU-A"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_GetSerialsBySKU_MissingSku(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.GetSerialsBySKU, "GET", "/serials/by-sku/", nil, gin.Params{{Key: "sku", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_CreateSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := requests.CreateSerialRequest{SerialNumber: "SN-NEW", SKU: "SKU-B"}
	w := performRequest(ctrl.CreateSerial, "POST", "/serials", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSerialsController_CreateSerial_InvalidJSON(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.CreateSerial, "POST", "/serials", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_UpdateSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := map[string]interface{}{"status": "used"}
	w := performRequest(ctrl.UpdateSerial, "PUT", "/serials/s1", body, gin.Params{{Key: "id", Value: "s1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_UpdateSerial_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := map[string]interface{}{"status": "used"}
	w := performRequest(ctrl.UpdateSerial, "PUT", "/serials/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_DeleteSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.DeleteSerial, "DELETE", "/serials/s1", nil, gin.Params{{Key: "id", Value: "s1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_DeleteSerial_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.DeleteSerial, "DELETE", "/serials/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSerialsController_TenantIsolation_forwardsControllerTenant verifies the
// constructor-injected TenantID is forwarded to every repo call.
func TestSerialsController_TenantIsolation_forwardsControllerTenant(t *testing.T) {
	repo := &mockSerialsRepoCtrl{}
	ctrl := newSerialsController(repo)

	performRequest(ctrl.GetSerialByID, "GET", "/serials/x", nil, gin.Params{{Key: "id", Value: "x"}})
	performRequest(ctrl.CreateSerial, "POST", "/serials", requests.CreateSerialRequest{SerialNumber: "SN-X", SKU: "SKU-X"}, nil)
	performRequest(ctrl.DeleteSerial, "DELETE", "/serials/x", nil, gin.Params{{Key: "id", Value: "x"}})

	for _, tid := range repo.gotTenantIDs {
		assert.Equal(t, ctrlTenantA, tid)
	}
	assert.NotContains(t, repo.gotTenantIDs, ctrlTenantB)
}
