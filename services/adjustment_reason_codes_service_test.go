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

// mockAdjustmentReasonCodesRepo is an in-memory fake for unit testing AdjustmentReasonCodesService.
type mockAdjustmentReasonCodesRepo struct {
	codes        []database.AdjustmentReasonCode
	byID         map[string]*database.AdjustmentReasonCode
	byCode       map[string]*database.AdjustmentReasonCode
	createResult *database.AdjustmentReasonCode
	createErr    *responses.InternalResponse
	updateResult *database.AdjustmentReasonCode
	updateErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
}

func (m *mockAdjustmentReasonCodesRepo) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return m.codes, nil
}

func (m *mockAdjustmentReasonCodesRepo) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return m.codes, nil
}

func (m *mockAdjustmentReasonCodesRepo) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.byID != nil {
		if rc, ok := m.byID[id]; ok {
			return rc, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "reason code no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockAdjustmentReasonCodesRepo) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.byCode != nil {
		if rc, ok := m.byCode[code]; ok {
			return rc, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "reason code no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockAdjustmentReasonCodesRepo) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createResult != nil {
		return m.createResult, nil
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	rc := &database.AdjustmentReasonCode{
		ID:        "rc-new",
		Code:      req.Code,
		Name:      req.Name,
		Direction: req.Direction,
		IsActive:  active,
	}
	return rc, nil
}

func (m *mockAdjustmentReasonCodesRepo) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updateResult != nil {
		return m.updateResult, nil
	}
	rc := &database.AdjustmentReasonCode{
		ID:        id,
		Code:      req.Code,
		Name:      req.Name,
		Direction: req.Direction,
	}
	return rc, nil
}

func (m *mockAdjustmentReasonCodesRepo) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

// --- Tests ---

func TestAdjustmentReasonCodesService_ListAdjustmentReasonCodes(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		codes: []database.AdjustmentReasonCode{
			{ID: "rc-1", Code: "RECEIVE", Name: "Recepción", Direction: "inbound", IsActive: true},
			{ID: "rc-2", Code: "DAMAGE", Name: "Daño", Direction: "outbound", IsActive: true},
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	list, errResp := svc.ListAdjustmentReasonCodes()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "RECEIVE", list[0].Code)
	assert.Equal(t, "DAMAGE", list[1].Code)
}

func TestAdjustmentReasonCodesService_ListAdjustmentReasonCodesAdmin(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		codes: []database.AdjustmentReasonCode{
			{ID: "rc-1", Code: "RECEIVE", Name: "Recepción", Direction: "inbound", IsActive: true},
			{ID: "rc-3", Code: "EXPIRED", Name: "Vencido", Direction: "outbound", IsActive: false},
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	list, errResp := svc.ListAdjustmentReasonCodesAdmin()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
}

func TestAdjustmentReasonCodesService_GetAdjustmentReasonCodeByID_Found(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		byID: map[string]*database.AdjustmentReasonCode{
			"rc-1": {ID: "rc-1", Code: "RECEIVE", Name: "Recepción", Direction: "inbound"},
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	rc, errResp := svc.GetAdjustmentReasonCodeByID("rc-1")
	require.Nil(t, errResp)
	require.NotNil(t, rc)
	assert.Equal(t, "RECEIVE", rc.Code)
	assert.Equal(t, "inbound", rc.Direction)
}

func TestAdjustmentReasonCodesService_GetAdjustmentReasonCodeByID_NotFound(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{byID: map[string]*database.AdjustmentReasonCode{}}
	svc := NewAdjustmentReasonCodesService(repo)
	rc, errResp := svc.GetAdjustmentReasonCodeByID("missing")
	require.NotNil(t, errResp)
	assert.Nil(t, rc)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAdjustmentReasonCodesService_GetAdjustmentReasonCodeByCode_Found(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		byCode: map[string]*database.AdjustmentReasonCode{
			"DAMAGE": {ID: "rc-2", Code: "DAMAGE", Name: "Daño", Direction: "outbound"},
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	rc, errResp := svc.GetAdjustmentReasonCodeByCode("DAMAGE")
	require.Nil(t, errResp)
	require.NotNil(t, rc)
	assert.Equal(t, "outbound", rc.Direction)
}

func TestAdjustmentReasonCodesService_GetAdjustmentReasonCodeByCode_NotFound(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{byCode: map[string]*database.AdjustmentReasonCode{}}
	svc := NewAdjustmentReasonCodesService(repo)
	rc, errResp := svc.GetAdjustmentReasonCodeByCode("UNKNOWN")
	require.NotNil(t, errResp)
	assert.Nil(t, rc)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAdjustmentReasonCodesService_CreateAdjustmentReasonCode_Success(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{}
	svc := NewAdjustmentReasonCodesService(repo)
	active := true
	req := &requests.AdjustmentReasonCodeCreate{
		Code:      "NEW-CODE",
		Name:      "New Reason",
		Direction: "inbound",
		IsActive:  &active,
	}
	rc, errResp := svc.CreateAdjustmentReasonCode(req)
	require.Nil(t, errResp)
	require.NotNil(t, rc)
	assert.Equal(t, "NEW-CODE", rc.Code)
	assert.Equal(t, "inbound", rc.Direction)
	assert.True(t, rc.IsActive)
}

func TestAdjustmentReasonCodesService_CreateAdjustmentReasonCode_Conflict(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		createErr: &responses.InternalResponse{
			Message:    "ya existe un reason code con ese código",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	req := &requests.AdjustmentReasonCodeCreate{Code: "DUP", Name: "Dup", Direction: "inbound"}
	rc, errResp := svc.CreateAdjustmentReasonCode(req)
	require.NotNil(t, errResp)
	assert.Nil(t, rc)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestAdjustmentReasonCodesService_UpdateAdjustmentReasonCode_Success(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{}
	svc := NewAdjustmentReasonCodesService(repo)
	req := &requests.AdjustmentReasonCodeUpdate{
		Code:      "RECEIVE",
		Name:      "Recepción Actualizada",
		Direction: "inbound",
	}
	rc, errResp := svc.UpdateAdjustmentReasonCode("rc-1", req)
	require.Nil(t, errResp)
	require.NotNil(t, rc)
	assert.Equal(t, "rc-1", rc.ID)
	assert.Equal(t, "Recepción Actualizada", rc.Name)
}

func TestAdjustmentReasonCodesService_UpdateAdjustmentReasonCode_NotFound(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		updateErr: &responses.InternalResponse{
			Message:    "reason code no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	req := &requests.AdjustmentReasonCodeUpdate{Code: "X", Name: "X", Direction: "inbound"}
	rc, errResp := svc.UpdateAdjustmentReasonCode("missing", req)
	require.NotNil(t, errResp)
	assert.Nil(t, rc)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAdjustmentReasonCodesService_DeleteAdjustmentReasonCode_Success(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{}
	svc := NewAdjustmentReasonCodesService(repo)
	errResp := svc.DeleteAdjustmentReasonCode("rc-1")
	require.Nil(t, errResp)
}

func TestAdjustmentReasonCodesService_DeleteAdjustmentReasonCode_Error(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "error al eliminar reason code",
			Handled: false,
		},
	}
	svc := NewAdjustmentReasonCodesService(repo)
	errResp := svc.DeleteAdjustmentReasonCode("rc-1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
