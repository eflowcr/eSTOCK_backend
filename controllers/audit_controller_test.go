package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock audit repo ──────────────────────────────────────────────────────────

type mockAuditRepo struct {
	entries  []ports.AuditLogEntry
	total    int64
	listErr  error
	countErr error
}

func (m *mockAuditRepo) Create(_ context.Context, _ ports.CreateAuditLogParams) error {
	return nil
}
func (m *mockAuditRepo) List(_ context.Context, _ ports.ListAuditLogsParams) ([]ports.AuditLogEntry, error) {
	return m.entries, m.listErr
}
func (m *mockAuditRepo) Count(_ context.Context, _ ports.ListAuditLogsParams) (int64, error) {
	return m.total, m.countErr
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestAuditController_ListAuditLogs_Success(t *testing.T) {
	repo := &mockAuditRepo{
		entries: []ports.AuditLogEntry{{ID: "log-1", Action: "create", ResourceType: "article"}},
		total:   1,
	}
	svc := services.NewAuditService(repo)
	ctrl := NewAuditController(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	c.Request = req
	ctrl.ListAuditLogs(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditController_ListAuditLogs_NilService(t *testing.T) {
	ctrl := NewAuditController(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	c.Request = req
	ctrl.ListAuditLogs(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditController_ListAuditLogs_Error(t *testing.T) {
	repo := &mockAuditRepo{listErr: errors.New("db error")}
	svc := services.NewAuditService(repo)
	ctrl := NewAuditController(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	c.Request = req
	ctrl.ListAuditLogs(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditController_ListAuditLogs_WithFilters(t *testing.T) {
	repo := &mockAuditRepo{entries: []ports.AuditLogEntry{}, total: 0}
	svc := services.NewAuditService(repo)
	ctrl := NewAuditController(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/audit-logs?page=2&per_page=10&action=create&resource_type=article", nil)
	c.Request = req
	ctrl.ListAuditLogs(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditController_ListAuditLogs_EmptyList(t *testing.T) {
	repo := &mockAuditRepo{entries: []ports.AuditLogEntry{}, total: 0}
	svc := services.NewAuditService(repo)
	ctrl := NewAuditController(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	c.Request = req
	ctrl.ListAuditLogs(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
