package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// mockLotsRepo is an in-memory fake for unit testing LotsService.
type mockLotsRepo struct {
	lots      []database.Lot
	getAllErr *responses.InternalResponse
	createErr *responses.InternalResponse
}

func (m *mockLotsRepo) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	if m.lots == nil {
		return nil, nil
	}
	return m.lots, nil
}

func (m *mockLotsRepo) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	if m.lots == nil {
		return nil, nil
	}
	if sku == nil || *sku == "" {
		return m.lots, nil
	}
	var out []database.Lot
	for _, l := range m.lots {
		if l.SKU == *sku {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockLotsRepo) CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse {
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockLotsRepo) UpdateLot(id int, data map[string]interface{}) *responses.InternalResponse {
	return nil
}

func (m *mockLotsRepo) DeleteLot(id int) *responses.InternalResponse {
	return nil
}

func TestLotsService_GetAllLots(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: 1, LotNumber: "L1", SKU: "SKU-A", Quantity: 10},
			{ID: 2, LotNumber: "L2", SKU: "SKU-B", Quantity: 20},
		},
	}
	svc := NewLotsService(repo)
	list, errResp := svc.GetAllLots()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "SKU-A", list[0].SKU)
	assert.Equal(t, "SKU-B", list[1].SKU)
}

func TestLotsService_GetLotsBySKU_NilSku_ReturnsAll(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: 1, SKU: "S1", Quantity: 1},
			{ID: 2, SKU: "S2", Quantity: 2},
		},
	}
	svc := NewLotsService(repo)
	list, errResp := svc.GetLotsBySKU(nil)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
}

func TestLotsService_GetLotsBySKU_Filtered(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: 1, SKU: "S1", Quantity: 1},
			{ID: 2, SKU: "S1", Quantity: 2},
			{ID: 3, SKU: "S2", Quantity: 3},
		},
	}
	svc := NewLotsService(repo)
	sku := "S1"
	list, errResp := svc.GetLotsBySKU(&sku)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "S1", list[0].SKU)
	assert.Equal(t, "S1", list[1].SKU)
}

func TestLotsService_CreateLot_Success(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo)
	req := &requests.CreateLotRequest{
		LotNumber: "LOT-001",
		SKU:       "ART-1",
		Quantity:  100,
	}
	errResp := svc.Create(req)
	require.Nil(t, errResp)
}

func TestLotsService_CreateLot_Error(t *testing.T) {
	repo := &mockLotsRepo{
		createErr: &responses.InternalResponse{
			Message: "Failed to create lot",
			Handled: false,
		},
	}
	svc := NewLotsService(repo)
	req := &requests.CreateLotRequest{LotNumber: "L", SKU: "S", Quantity: 1}
	errResp := svc.Create(req)
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
