package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInventoryMovementsRepo is an in-memory fake for unit testing InventoryMovementsService.
type mockInventoryMovementsRepo struct {
	movements []database.InventoryMovement
	err       *responses.InternalResponse
}

func (m *mockInventoryMovementsRepo) GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	return m.movements, m.err
}

func (m *mockInventoryMovementsRepo) ListMovements(_ ports.MovementsFilter) ([]database.InventoryMovement, *responses.InternalResponse) {
	return m.movements, m.err
}

func TestInventoryMovementsService_GetAllInventoryMovements_Success(t *testing.T) {
	reason := "Receiving task"
	movements := []database.InventoryMovement{
		{
			ID:             "mov-1",
			SKU:            "SKU1",
			Location:       "A-01",
			MovementType:   "inbound",
			Quantity:       50,
			RemainingStock: 50,
			Reason:         &reason,
			CreatedBy:      "user-1",
		},
		{
			ID:             "mov-2",
			SKU:            "SKU1",
			Location:       "A-01",
			MovementType:   "outbound",
			Quantity:       10,
			RemainingStock: 40,
			CreatedBy:      "user-2",
		},
	}
	repo := &mockInventoryMovementsRepo{movements: movements}
	svc := NewInventoryMovementsService(repo)

	result, errResp := svc.GetAllInventoryMovements("SKU1")
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.Equal(t, "SKU1", result[0].SKU)
	assert.Equal(t, "inbound", result[0].MovementType)
	assert.Equal(t, float64(50), result[0].Quantity)
	assert.Equal(t, "outbound", result[1].MovementType)
	assert.Equal(t, float64(40), result[1].RemainingStock)
}

func TestInventoryMovementsService_GetAllInventoryMovements_Empty(t *testing.T) {
	repo := &mockInventoryMovementsRepo{movements: []database.InventoryMovement{}}
	svc := NewInventoryMovementsService(repo)

	result, errResp := svc.GetAllInventoryMovements("SKU-NONE")
	require.Nil(t, errResp)
	assert.Empty(t, result)
}

func TestInventoryMovementsService_GetAllInventoryMovements_Error(t *testing.T) {
	repo := &mockInventoryMovementsRepo{
		err: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching inventory movements",
			Handled: false,
		},
	}
	svc := NewInventoryMovementsService(repo)

	result, errResp := svc.GetAllInventoryMovements("SKU1")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestInventoryMovementsService_GetAllInventoryMovements_NotFound(t *testing.T) {
	repo := &mockInventoryMovementsRepo{
		err: &responses.InternalResponse{
			Message:    "SKU not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewInventoryMovementsService(repo)

	result, errResp := svc.GetAllInventoryMovements("SKU-MISSING")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}
