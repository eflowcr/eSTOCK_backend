package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockAdjustmentsRepoCtrl struct {
	adjustments []database.Adjustment
	byID        map[string]*database.Adjustment
	details     map[string]*dto.AdjustmentDetails
	createErr   *responses.InternalResponse
	exportData  []byte
	exportErr   *responses.InternalResponse
}

func (m *mockAdjustmentsRepoCtrl) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return m.adjustments, nil
}

func (m *mockAdjustmentsRepoCtrl) GetAllForTenant(tenantID string) ([]database.Adjustment, *responses.InternalResponse) {
	return m.adjustments, nil
}

func (m *mockAdjustmentsRepoCtrl) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	if m.byID != nil {
		if a, ok := m.byID[id]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockAdjustmentsRepoCtrl) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	if m.details != nil {
		if d, ok := m.details[id]; ok {
			return d, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockAdjustmentsRepoCtrl) CreateAdjustment(userId string, tenantID string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.Adjustment{ID: "new-adj-1", SKU: adjustment.SKU, Location: adjustment.Location, Reason: adjustment.Reason, UserID: userId}, nil
}

func (m *mockAdjustmentsRepoCtrl) ExportAdjustmentsToExcel(_ string) ([]byte, *responses.InternalResponse) {
	return m.exportData, m.exportErr
}

func (m *mockAdjustmentsRepoCtrl) GetInventoryForAdjustment(sku, location string) (*database.Inventory, *responses.InternalResponse) {
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

// mock reason codes repo (used by AdjustmentsService)
type mockAdjustmentReasonCodesRepoForAdj struct {
	byCode map[string]*database.AdjustmentReasonCode
}

func (m *mockAdjustmentReasonCodesRepoForAdj) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.byCode != nil {
		if rc, ok := m.byCode[code]; ok {
			return rc, nil
		}
	}
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentReasonCodesRepoForAdj) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newAdjustmentsController(repo *mockAdjustmentsRepoCtrl, rcRepo *mockAdjustmentReasonCodesRepoForAdj) *AdjustmentsController {
	svc := services.NewAdjustmentsService(repo, rcRepo)
	return NewAdjustmentsController(*svc, testJWTSecret, nil)
}

func defaultRCRepo() *mockAdjustmentReasonCodesRepoForAdj {
	return &mockAdjustmentReasonCodesRepoForAdj{
		byCode: map[string]*database.AdjustmentReasonCode{
			"INBOUND": {ID: "rc-1", Code: "INBOUND", Name: "Inbound", Direction: "inbound", IsActive: true},
		},
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestAdjustmentsController_GetAllAdjustments_Empty(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{adjustments: []database.Adjustment{}}, defaultRCRepo())
	w := performRequest(ctrl.GetAllAdjustments, "GET", "/adjustments", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_GetAllAdjustments_WithData(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{
		adjustments: []database.Adjustment{
			{ID: "adj-1", SKU: "SKU-001", Location: "A01", Reason: "INBOUND", UserID: "user-1"},
		},
	}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	w := performRequest(ctrl.GetAllAdjustments, "GET", "/adjustments", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_GetAdjustmentByID_Found(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{
		byID: map[string]*database.Adjustment{
			"adj-1": {ID: "adj-1", SKU: "SKU-001", Location: "A01", Reason: "INBOUND", UserID: "user-1"},
		},
	}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentByID, "GET", "/adjustments/adj-1", nil, gin.Params{{Key: "id", Value: "adj-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_GetAdjustmentByID_NotFound(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{byID: map[string]*database.Adjustment{}}, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentByID, "GET", "/adjustments/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdjustmentsController_GetAdjustmentByID_MissingParam(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{}, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentByID, "GET", "/adjustments/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentsController_GetAdjustmentDetails_Found(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{
		details: map[string]*dto.AdjustmentDetails{
			"adj-1": {Adjustment: database.Adjustment{ID: "adj-1"}},
		},
	}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentDetails, "GET", "/adjustments/adj-1/details", nil, gin.Params{{Key: "id", Value: "adj-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_GetAdjustmentDetails_NotFound(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{details: map[string]*dto.AdjustmentDetails{}}, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentDetails, "GET", "/adjustments/99/details", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdjustmentsController_GetAdjustmentDetails_MissingParam(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{}, defaultRCRepo())
	w := performRequest(ctrl.GetAdjustmentDetails, "GET", "/adjustments//details", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentsController_CreateAdjustment_Success(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{}, defaultRCRepo())
	body := requests.CreateAdjustment{
		SKU:                "SKU-001",
		Location:           "A01",
		AdjustmentQuantity: 10,
		Reason:             "INBOUND",
	}
	w := performRequestWithHeader(ctrl.CreateAdjustment, "POST", "/adjustments", body, nil, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAdjustmentsController_CreateAdjustment_InvalidJSON(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{}, defaultRCRepo())
	w := performRequest(ctrl.CreateAdjustment, "POST", "/adjustments", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentsController_CreateAdjustment_Unauthorized(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{}, defaultRCRepo())
	body := requests.CreateAdjustment{
		SKU:                "SKU-001",
		Location:           "A01",
		AdjustmentQuantity: 10,
		Reason:             "INBOUND",
	}
	// No token provided
	w := performRequest(ctrl.CreateAdjustment, "POST", "/adjustments", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdjustmentsController_CreateAdjustment_ServiceError(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	body := requests.CreateAdjustment{
		SKU:                "SKU-001",
		Location:           "A01",
		AdjustmentQuantity: 10,
		Reason:             "INBOUND",
	}
	w := performRequestWithHeader(ctrl.CreateAdjustment, "POST", "/adjustments", body, nil, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdjustmentsController_ExportAdjustmentsToExcel_Success(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{exportData: []byte("xlsx-data")}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	w := performRequest(ctrl.ExportAdjustmentsToExcel, "GET", "/adjustments/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_ExportAdjustmentsToExcel_NoData(t *testing.T) {
	ctrl := newAdjustmentsController(&mockAdjustmentsRepoCtrl{exportData: nil}, defaultRCRepo())
	w := performRequest(ctrl.ExportAdjustmentsToExcel, "GET", "/adjustments/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentsController_ExportAdjustmentsToExcel_Error(t *testing.T) {
	repo := &mockAdjustmentsRepoCtrl{
		exportErr: &responses.InternalResponse{
			Message:    "export failed",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newAdjustmentsController(repo, defaultRCRepo())
	w := performRequest(ctrl.ExportAdjustmentsToExcel, "GET", "/adjustments/export", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
