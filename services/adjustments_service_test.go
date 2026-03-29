package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAdjustmentsRepo is an in-memory fake for unit testing AdjustmentsService.
type mockAdjustmentsRepo struct {
	adjustments  []database.Adjustment
	byID         map[string]*database.Adjustment
	detailsByID  map[string]*dto.AdjustmentDetails
	createResult *database.Adjustment
	createErr    *responses.InternalResponse
	exportBytes  []byte
	exportErr    *responses.InternalResponse
}

func (m *mockAdjustmentsRepo) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return m.adjustments, nil
}

func (m *mockAdjustmentsRepo) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	if m.byID != nil {
		if a, ok := m.byID[id]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "ajuste no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockAdjustmentsRepo) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	if m.detailsByID != nil {
		if d, ok := m.detailsByID[id]; ok {
			return d, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "ajuste no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockAdjustmentsRepo) CreateAdjustment(userId string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createResult != nil {
		return m.createResult, nil
	}
	result := &database.Adjustment{
		ID:            "adj-new",
		SKU:           adjustment.SKU,
		Location:      adjustment.Location,
		AdjustmentQty: int(adjustment.AdjustmentQuantity),
		Reason:        adjustment.Reason,
		UserID:        userId,
	}
	return result, nil
}

func (m *mockAdjustmentsRepo) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	return m.exportBytes, m.exportErr
}

// mockReasonCodesRepo is an in-memory fake for AdjustmentReasonCodesRepository used by AdjustmentsService.
type mockReasonCodesForAdjRepo struct {
	byCode    map[string]*database.AdjustmentReasonCode
	lookupErr *responses.InternalResponse
}

func (m *mockReasonCodesForAdjRepo) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.lookupErr != nil {
		return nil, m.lookupErr
	}
	if m.byCode != nil {
		if rc, ok := m.byCode[code]; ok {
			return rc, nil
		}
	}
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockReasonCodesForAdjRepo) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	return nil
}

// --- Tests ---

func TestAdjustmentsService_GetAllAdjustments(t *testing.T) {
	repo := &mockAdjustmentsRepo{
		adjustments: []database.Adjustment{
			{ID: "a1", SKU: "SKU-1", Location: "LOC-A"},
			{ID: "a2", SKU: "SKU-2", Location: "LOC-B"},
		},
	}
	svc := NewAdjustmentsService(repo, nil)
	list, errResp := svc.GetAllAdjustments()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "SKU-1", list[0].SKU)
}

func TestAdjustmentsService_GetAdjustmentByID_Found(t *testing.T) {
	repo := &mockAdjustmentsRepo{
		byID: map[string]*database.Adjustment{
			"a1": {ID: "a1", SKU: "SKU-1", Location: "LOC-A"},
		},
	}
	svc := NewAdjustmentsService(repo, nil)
	adj, errResp := svc.GetAdjustmentByID("a1")
	require.Nil(t, errResp)
	require.NotNil(t, adj)
	assert.Equal(t, "SKU-1", adj.SKU)
}

func TestAdjustmentsService_GetAdjustmentByID_NotFound(t *testing.T) {
	repo := &mockAdjustmentsRepo{byID: map[string]*database.Adjustment{}}
	svc := NewAdjustmentsService(repo, nil)
	adj, errResp := svc.GetAdjustmentByID("missing")
	require.NotNil(t, errResp)
	assert.Nil(t, adj)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAdjustmentsService_GetAdjustmentDetails_Found(t *testing.T) {
	repo := &mockAdjustmentsRepo{
		detailsByID: map[string]*dto.AdjustmentDetails{
			"a1": {Adjustment: database.Adjustment{ID: "a1", SKU: "SKU-1"}},
		},
	}
	svc := NewAdjustmentsService(repo, nil)
	details, errResp := svc.GetAdjustmentDetails("a1")
	require.Nil(t, errResp)
	require.NotNil(t, details)
	assert.Equal(t, "SKU-1", details.Adjustment.SKU)
}

func TestAdjustmentsService_GetAdjustmentDetails_NotFound(t *testing.T) {
	repo := &mockAdjustmentsRepo{detailsByID: map[string]*dto.AdjustmentDetails{}}
	svc := NewAdjustmentsService(repo, nil)
	details, errResp := svc.GetAdjustmentDetails("missing")
	require.NotNil(t, errResp)
	assert.Nil(t, details)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAdjustmentsService_CreateAdjustment_NegativeQuantity_ReturnsBadRequest(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	svc := NewAdjustmentsService(repo, nil)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: -5,
		Reason:             "DAMAGE",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

func TestAdjustmentsService_CreateAdjustment_NoReasonCodesRepo_Success(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	svc := NewAdjustmentsService(repo, nil)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: 10,
		Reason:             "RECOUNT",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "SKU-1", result.SKU)
	assert.Equal(t, 10, result.AdjustmentQty)
}

func TestAdjustmentsService_CreateAdjustment_InboundReasonCode_KeepsPositiveQuantity(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	rcRepo := &mockReasonCodesForAdjRepo{
		byCode: map[string]*database.AdjustmentReasonCode{
			"RECEIVE": {ID: "rc-1", Code: "RECEIVE", Direction: "inbound"},
		},
	}
	svc := NewAdjustmentsService(repo, rcRepo)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: 20,
		Reason:             "RECEIVE",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	// inbound: quantity stays positive
	assert.Equal(t, 20, result.AdjustmentQty)
}

func TestAdjustmentsService_CreateAdjustment_OutboundReasonCode_NegatesQuantity(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	rcRepo := &mockReasonCodesForAdjRepo{
		byCode: map[string]*database.AdjustmentReasonCode{
			"DAMAGE": {ID: "rc-2", Code: "DAMAGE", Direction: "outbound"},
		},
	}
	svc := NewAdjustmentsService(repo, rcRepo)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: 15,
		Reason:             "DAMAGE",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	// outbound: quantity is negated
	assert.Equal(t, -15, result.AdjustmentQty)
}

func TestAdjustmentsService_CreateAdjustment_InvalidReasonCode_ReturnsBadRequest(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	rcRepo := &mockReasonCodesForAdjRepo{
		byCode: map[string]*database.AdjustmentReasonCode{},
	}
	svc := NewAdjustmentsService(repo, rcRepo)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: 5,
		Reason:             "UNKNOWN-CODE",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestAdjustmentsService_CreateAdjustment_ReasonCodeLookupError_ReturnsError(t *testing.T) {
	repo := &mockAdjustmentsRepo{}
	rcRepo := &mockReasonCodesForAdjRepo{
		lookupErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "error al obtener reason code",
			Handled: false,
		},
	}
	svc := NewAdjustmentsService(repo, rcRepo)
	req := requests.CreateAdjustment{
		SKU:                "SKU-1",
		Location:           "LOC-A",
		AdjustmentQuantity: 5,
		Reason:             "SOME-CODE",
	}
	result, errResp := svc.CreateAdjustment("user-1", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestAdjustmentsService_ExportAdjustmentsToExcel_Success(t *testing.T) {
	repo := &mockAdjustmentsRepo{exportBytes: []byte("excel-data")}
	svc := NewAdjustmentsService(repo, nil)
	data, errResp := svc.ExportAdjustmentsToExcel()
	require.Nil(t, errResp)
	assert.Equal(t, []byte("excel-data"), data)
}

func TestAdjustmentsService_ExportAdjustmentsToExcel_Error(t *testing.T) {
	repo := &mockAdjustmentsRepo{
		exportErr: &responses.InternalResponse{
			Error:   errors.New("export failed"),
			Message: "error al exportar",
			Handled: false,
		},
	}
	svc := NewAdjustmentsService(repo, nil)
	data, errResp := svc.ExportAdjustmentsToExcel()
	require.NotNil(t, errResp)
	assert.Nil(t, data)
}
