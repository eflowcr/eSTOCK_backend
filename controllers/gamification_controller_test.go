package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockGamificationRepoCtrl struct {
	userStat         *database.UserStat
	userStatErr      *responses.InternalResponse
	badges           []database.Badge
	badgesErr        *responses.InternalResponse
	allBadges        []database.Badge
	allBadgesErr     *responses.InternalResponse
	userBadges       []database.UserBadge
	completeTasksErr *responses.InternalResponse
	allStats         []responses.UserStatsResponse
	allStatsErr      *responses.InternalResponse
}

func (m *mockGamificationRepoCtrl) GamificationStats(userId string) (*database.UserStat, *responses.InternalResponse) {
	return m.userStat, m.userStatErr
}

func (m *mockGamificationRepoCtrl) Badges(userId string) ([]database.Badge, *responses.InternalResponse) {
	return m.badges, m.badgesErr
}

func (m *mockGamificationRepoCtrl) GetAllBadges() ([]database.Badge, *responses.InternalResponse) {
	return m.allBadges, m.allBadgesErr
}

func (m *mockGamificationRepoCtrl) CompleteTasks(userId string, task requests.CompleteTasks) ([]database.UserBadge, *responses.InternalResponse) {
	return m.userBadges, m.completeTasksErr
}

func (m *mockGamificationRepoCtrl) GetAllStats() ([]responses.UserStatsResponse, *responses.InternalResponse) {
	return m.allStats, m.allStatsErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newGamificationController(repo *mockGamificationRepoCtrl) *GamificationController {
	svc := services.NewGamificationService(repo)
	return NewGamificationController(*svc, testJWTSecret)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestGamificationController_GamificationStats_Success(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		userStat: &database.UserStat{ID: "stat-1", UserID: "user-1", ReceivingTasksCompleted: 5},
	}
	ctrl := newGamificationController(repo)
	w := performRequestWithHeader(ctrl.GamificationStats, "GET", "/gamification/stats", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

func TestGamificationController_GamificationStats_NotFound(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{userStat: nil})
	w := performRequestWithHeader(ctrl.GamificationStats, "GET", "/gamification/stats", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGamificationController_GamificationStats_Unauthorized(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{})
	w := performRequest(ctrl.GamificationStats, "GET", "/gamification/stats", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGamificationController_GamificationStats_ServiceError(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		userStatErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newGamificationController(repo)
	w := performRequestWithHeader(ctrl.GamificationStats, "GET", "/gamification/stats", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGamificationController_Badges_Success(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		badges: []database.Badge{{ID: "badge-1", Name: "First Pick", Description: "Completed first pick", Emoji: "🏅"}},
	}
	ctrl := newGamificationController(repo)
	w := performRequestWithHeader(ctrl.Badges, "GET", "/gamification/badges", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_Badges_NotFound(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{badges: nil})
	w := performRequestWithHeader(ctrl.Badges, "GET", "/gamification/badges", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGamificationController_Badges_Unauthorized(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{})
	w := performRequest(ctrl.Badges, "GET", "/gamification/badges", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGamificationController_Badges_ServiceError(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		badgesErr: &responses.InternalResponse{
			Message:    "internal error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newGamificationController(repo)
	w := performRequestWithHeader(ctrl.Badges, "GET", "/gamification/badges", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGamificationController_GetAllBadges_Success(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		allBadges: []database.Badge{{ID: "badge-1", Name: "First Pick", Description: "Completed first pick", Emoji: "🏅"}},
	}
	ctrl := newGamificationController(repo)
	w := performRequest(ctrl.GetAllBadges, "GET", "/gamification/all-badges", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_GetAllBadges_Empty(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{allBadges: nil})
	w := performRequest(ctrl.GetAllBadges, "GET", "/gamification/all-badges", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_GetAllBadges_ServiceError(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		allBadgesErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newGamificationController(repo)
	w := performRequest(ctrl.GetAllBadges, "GET", "/gamification/all-badges", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGamificationController_CompleteTasks_Success(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		userBadges: []database.UserBadge{{ID: "ub-1", UserID: "user-1", BadgeID: "badge-1"}},
	}
	ctrl := newGamificationController(repo)
	body := requests.CompleteTasks{TaskType: "receiving", CompletionTime: 120}
	w := performRequestWithHeader(ctrl.CompleteTasks, "POST", "/gamification/complete-tasks", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_CompleteTasks_NoNewBadges(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{userBadges: nil})
	body := requests.CompleteTasks{TaskType: "receiving", CompletionTime: 120}
	w := performRequestWithHeader(ctrl.CompleteTasks, "POST", "/gamification/complete-tasks", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_CompleteTasks_Unauthorized(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{})
	body := requests.CompleteTasks{TaskType: "receiving", CompletionTime: 120}
	w := performRequest(ctrl.CompleteTasks, "POST", "/gamification/complete-tasks", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGamificationController_CompleteTasks_InvalidJSON(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompleteTasks, "POST", "/gamification/complete-tasks", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGamificationController_CompleteTasks_ServiceError(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		completeTasksErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newGamificationController(repo)
	body := requests.CompleteTasks{TaskType: "picking", CompletionTime: 60}
	w := performRequestWithHeader(ctrl.CompleteTasks, "POST", "/gamification/complete-tasks", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGamificationController_GetAllUserStats_Success(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		allStats: []responses.UserStatsResponse{{ID: "stat-1", UserID: "user-1", Username: "testuser"}},
	}
	ctrl := newGamificationController(repo)
	w := performRequest(ctrl.GetAllUserStats, "GET", "/gamification/all-stats", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_GetAllUserStats_Empty(t *testing.T) {
	ctrl := newGamificationController(&mockGamificationRepoCtrl{allStats: nil})
	w := performRequest(ctrl.GetAllUserStats, "GET", "/gamification/all-stats", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGamificationController_GetAllUserStats_ServiceError(t *testing.T) {
	repo := &mockGamificationRepoCtrl{
		allStatsErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newGamificationController(repo)
	w := performRequest(ctrl.GetAllUserStats, "GET", "/gamification/all-stats", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
