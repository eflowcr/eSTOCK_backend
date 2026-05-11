package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIdempotencyRepo is a thread-safe in-memory mock used across the
// middleware test matrix. Each test instantiates its own to avoid bleed.
type fakeIdempotencyRepo struct {
	mu      sync.Mutex
	rows    map[string]*database.IdempotencyKey // composite key = key + "|" + userID
	lookErr error
}

func newFakeRepo() *fakeIdempotencyRepo {
	return &fakeIdempotencyRepo{rows: map[string]*database.IdempotencyKey{}}
}

func (f *fakeIdempotencyRepo) Lookup(_ context.Context, key, userID string) (*database.IdempotencyKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.lookErr != nil {
		return nil, f.lookErr
	}
	row, ok := f.rows[key+"|"+userID]
	if !ok {
		return nil, nil
	}
	if row.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	return row, nil
}

func (f *fakeIdempotencyRepo) Store(_ context.Context, entry *database.IdempotencyKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// ON CONFLICT DO NOTHING semantics — first writer wins.
	composite := entry.Key + "|" + entry.UserID
	if _, exists := f.rows[composite]; exists {
		return nil
	}
	f.rows[composite] = entry
	return nil
}

func (f *fakeIdempotencyRepo) SweepExpired(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for k, row := range f.rows {
		if row.ExpiresAt.Before(time.Now()) {
			delete(f.rows, k)
			n++
		}
	}
	return n, nil
}

// runRequest fires a single request through a router with JWT + idempotency
// middleware in front. handlerCalls counts invocations so tests can assert
// replays short-circuit the handler.
func runRequest(t *testing.T, repo *fakeIdempotencyRepo, key, body string, handler gin.HandlerFunc) (*httptest.ResponseRecorder, *int) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	calls := 0
	wrapped := gin.HandlerFunc(func(c *gin.Context) {
		calls++
		handler(c)
	})

	r := gin.New()
	// Fake auth — set user_id directly.
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware(repo))
	r.POST("/api/mobile/picking-tasks/:id/complete-line", wrapped)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/mobile/picking-tasks/abc123/complete-line", nil)
	if body != "" {
		req.Body = http.NoBody // request body not needed for middleware behavior
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	r.ServeHTTP(w, req)
	return w, &calls
}

func TestIdempotencyMiddleware_NoHeaderPassthrough(t *testing.T) {
	repo := newFakeRepo()
	w, calls := runRequest(t, repo, "", "", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, *calls, "handler must run when header absent")
	assert.Empty(t, repo.rows, "no cache entry when header absent")
}

func TestIdempotencyMiddleware_FirstHitStoresResponse(t *testing.T) {
	repo := newFakeRepo()
	w, calls := runRequest(t, repo, "key-abc", "", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"line_id": "L1", "picked_qty": 5})
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, *calls)
	require.Len(t, repo.rows, 1, "cache must contain the response after first hit")
	row := repo.rows["key-abc|user-1"]
	assert.Equal(t, http.StatusOK, row.ResponseStatus)
	assert.Contains(t, string(row.ResponseBody), `"line_id":"L1"`)
}

func TestIdempotencyMiddleware_ReplayReturnsCachedShortCircuit(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["key-abc|user-1"] = &database.IdempotencyKey{
		Key:            "key-abc",
		UserID:         "user-1",
		ResponseStatus: http.StatusOK,
		ResponseBody:   []byte(`{"replayed":true}`),
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	w, calls := runRequest(t, repo, "key-abc", "", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"should_not_fire": true})
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, *calls, "handler must NOT run on replay")
	assert.JSONEq(t, `{"replayed":true}`, w.Body.String())
	assert.Equal(t, "true", w.Header().Get("X-Idempotent-Replayed"))
}

func TestIdempotencyMiddleware_ExpiredEntryRunsHandler(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["key-abc|user-1"] = &database.IdempotencyKey{
		Key:            "key-abc",
		UserID:         "user-1",
		ResponseStatus: http.StatusOK,
		ResponseBody:   []byte(`{"stale":true}`),
		ExpiresAt:      time.Now().Add(-1 * time.Hour),
	}

	w, calls := runRequest(t, repo, "key-abc", "", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"fresh": true})
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, *calls, "handler must run when cache entry is expired")
	assert.Contains(t, w.Body.String(), `"fresh":true`)
}

func TestIdempotencyMiddleware_5xxNotCached(t *testing.T) {
	repo := newFakeRepo()
	w, calls := runRequest(t, repo, "key-abc", "", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transient"})
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1, *calls)
	assert.Empty(t, repo.rows, "5xx responses must not be cached — retry should re-attempt")
}

func TestIdempotencyMiddleware_4xxCached(t *testing.T) {
	// Validation rejections SHOULD be cached: replay returns the same 4xx
	// instead of re-running validation. This is the desired idempotent
	// semantic — same input, same rejection.
	repo := newFakeRepo()
	w, calls := runRequest(t, repo, "key-abc", "", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_qty"})
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 1, *calls)
	require.Len(t, repo.rows, 1, "4xx responses cached so replays surface the same rejection")
}

func TestIdempotencyMiddleware_NilRepoPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, "user-1")
		c.Next()
	})
	r.Use(IdempotencyMiddleware(nil))
	r.POST("/x", func(c *gin.Context) {
		calls++
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	req.Header.Set("Idempotency-Key", "key-abc")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, calls)
}

func TestIdempotencyMiddleware_MissingUserIDPassthrough(t *testing.T) {
	// If JWT middleware didn't run or set ContextKeyUserID, the idempotency
	// middleware should pass through rather than 500.
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo()
	calls := 0
	r := gin.New()
	r.Use(IdempotencyMiddleware(repo))
	r.POST("/x", func(c *gin.Context) {
		calls++
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	req.Header.Set("Idempotency-Key", "key-abc")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, calls)
	assert.Empty(t, repo.rows, "no caching without user scope")
}
