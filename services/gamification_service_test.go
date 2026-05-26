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

// mockGamificationRepo is an in-memory fake for unit testing GamificationService.
type mockGamificationRepo struct {
	userStat        *database.UserStat
	userStatErr     *responses.InternalResponse
	badges          []database.Badge
	badgesErr       *responses.InternalResponse
	allBadges       []database.Badge
	allBadgesErr    *responses.InternalResponse
	userBadges      []database.UserBadge
	completeTaskErr *responses.InternalResponse
	allStats        []responses.UserStatsResponse
	allStatsErr     *responses.InternalResponse
}

func (m *mockGamificationRepo) GamificationStats(userId string) (*database.UserStat, *responses.InternalResponse) {
	return m.userStat, m.userStatErr
}

func (m *mockGamificationRepo) Badges(userId string) ([]database.Badge, *responses.InternalResponse) {
	return m.badges, m.badgesErr
}

func (m *mockGamificationRepo) GetAllBadges() ([]database.Badge, *responses.InternalResponse) {
	return m.allBadges, m.allBadgesErr
}

func (m *mockGamificationRepo) CompleteTasks(userId string, task requests.CompleteTasks) ([]database.UserBadge, *responses.InternalResponse) {
	return m.userBadges, m.completeTaskErr
}

func (m *mockGamificationRepo) GetAllStats() ([]responses.UserStatsResponse, *responses.InternalResponse) {
	return m.allStats, m.allStatsErr
}

func TestGamificationService_GamificationStats_Success(t *testing.T) {
	stat := &database.UserStat{
		ID:                    "stat-1",
		UserID:                "user-1",
		PickingTasksCompleted: 10,
	}
	repo := &mockGamificationRepo{userStat: stat}
	svc := NewGamificationService(repo)

	result, errResp := svc.GamificationStats("user-1")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "user-1", result.UserID)
	assert.Equal(t, 10, result.PickingTasksCompleted)
}

func TestGamificationService_GamificationStats_NotFound(t *testing.T) {
	repo := &mockGamificationRepo{
		userStatErr: &responses.InternalResponse{
			Message:    "Stats not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewGamificationService(repo)

	result, errResp := svc.GamificationStats("user-99")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

func TestGamificationService_Badges_Success(t *testing.T) {
	badges := []database.Badge{
		{ID: "badge-1", Name: "First Pick", RuleType: "picking"},
		{ID: "badge-2", Name: "Speed Demon", RuleType: "speed"},
	}
	repo := &mockGamificationRepo{badges: badges}
	svc := NewGamificationService(repo)

	result, errResp := svc.Badges("user-1")
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.Equal(t, "badge-1", result[0].ID)
	assert.Equal(t, "Speed Demon", result[1].Name)
}

func TestGamificationService_Badges_Error(t *testing.T) {
	repo := &mockGamificationRepo{
		badgesErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching badges",
			Handled: false,
		},
	}
	svc := NewGamificationService(repo)

	result, errResp := svc.Badges("user-1")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestGamificationService_GetAllBadges_Success(t *testing.T) {
	badges := []database.Badge{
		{ID: "badge-1", Name: "First Pick"},
		{ID: "badge-2", Name: "Speed Demon"},
		{ID: "badge-3", Name: "Master Receiver"},
	}
	repo := &mockGamificationRepo{allBadges: badges}
	svc := NewGamificationService(repo)

	result, errResp := svc.GetAllBadges()
	require.Nil(t, errResp)
	require.Len(t, result, 3)
	assert.Equal(t, "badge-3", result[2].ID)
}

func TestGamificationService_GetAllBadges_Empty(t *testing.T) {
	repo := &mockGamificationRepo{allBadges: []database.Badge{}}
	svc := NewGamificationService(repo)

	result, errResp := svc.GetAllBadges()
	require.Nil(t, errResp)
	assert.Empty(t, result)
}

func TestGamificationService_CompleteTasks_Success(t *testing.T) {
	accuracy := 95
	userBadges := []database.UserBadge{
		{ID: "ub-1", UserID: "user-1", BadgeID: "badge-1"},
	}
	repo := &mockGamificationRepo{userBadges: userBadges}
	svc := NewGamificationService(repo)

	task := requests.CompleteTasks{
		TaskType:       "picking",
		CompletionTime: 120,
		Accuracy:       &accuracy,
	}
	result, errResp := svc.CompleteTasks("user-1", task)
	require.Nil(t, errResp)
	require.Len(t, result, 1)
	assert.Equal(t, "badge-1", result[0].BadgeID)
}

func TestGamificationService_CompleteTasks_Error(t *testing.T) {
	repo := &mockGamificationRepo{
		completeTaskErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error completing task",
			Handled: false,
		},
	}
	svc := NewGamificationService(repo)

	task := requests.CompleteTasks{TaskType: "picking", CompletionTime: 60}
	result, errResp := svc.CompleteTasks("user-1", task)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestGamificationService_GetAllUserStats_Success(t *testing.T) {
	stats := []responses.UserStatsResponse{
		{ID: "stat-1", UserID: "user-1", Username: "alice", PickingTasksCompleted: 5},
		{ID: "stat-2", UserID: "user-2", Username: "bob", PickingTasksCompleted: 3},
	}
	repo := &mockGamificationRepo{allStats: stats}
	svc := NewGamificationService(repo)

	result, errResp := svc.GetAllUserStats()
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.Equal(t, "alice", result[0].Username)
	assert.Equal(t, "bob", result[1].Username)
}

func TestGamificationService_GetAllUserStats_Error(t *testing.T) {
	repo := &mockGamificationRepo{
		allStatsErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching stats",
			Handled: false,
		},
	}
	svc := NewGamificationService(repo)

	result, errResp := svc.GetAllUserStats()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}
