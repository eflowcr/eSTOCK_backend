package tools

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// newRateLimitTestRouter wires the rate limit middleware to a /ping handler
// from a fixed client IP (set via X-Forwarded-For requires TrustedProxies setup,
// so we just rely on Gin's RemoteAddr-based ClientIP).
func newRateLimitTestRouter(t *testing.T, r rate.Limit, b int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewIPRateLimiter(r, b))
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return router
}

func doRateLimitedReq(router *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	router.ServeHTTP(w, req)
	return w
}

func TestRateLimit_ResponseIncludesXRateLimitHeaders(t *testing.T) {
	// 5 tokens burst, very slow refill so token count stays predictable.
	router := newRateLimitTestRouter(t, rate.Every(time.Hour), 5)

	w := doRateLimitedReq(router)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on first request, got %d", w.Code)
	}
	if got := w.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Fatalf("expected X-RateLimit-Limit=5, got %q", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got == "" {
		t.Fatalf("expected X-RateLimit-Remaining present, got empty")
	}
	if got := w.Header().Get("X-RateLimit-Reset"); got == "" {
		t.Fatalf("expected X-RateLimit-Reset present, got empty")
	}
	resetUnix, err := strconv.ParseInt(w.Header().Get("X-RateLimit-Reset"), 10, 64)
	if err != nil {
		t.Fatalf("X-RateLimit-Reset must be integer unix ts, got %v: %v", w.Header().Get("X-RateLimit-Reset"), err)
	}
	if resetUnix < time.Now().Unix() {
		t.Fatalf("X-RateLimit-Reset should be in the future or now, got %d (now=%d)", resetUnix, time.Now().Unix())
	}
}

func TestRateLimit_429IncludesRetryAfter(t *testing.T) {
	// burst=2 so we exhaust quickly.
	router := newRateLimitTestRouter(t, rate.Every(time.Hour), 2)
	// Drain bucket.
	_ = doRateLimitedReq(router)
	_ = doRateLimitedReq(router)
	// Third request should 429.
	w := doRateLimitedReq(router)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on exhaustion, got %d", w.Code)
	}
	ra := w.Header().Get("Retry-After")
	if ra == "" {
		t.Fatalf("expected Retry-After header on 429, got empty")
	}
	secs, err := strconv.Atoi(ra)
	if err != nil {
		t.Fatalf("Retry-After must be integer seconds (RFC 6585), got %q: %v", ra, err)
	}
	if secs < 1 {
		t.Fatalf("Retry-After must be >= 1 second, got %d", secs)
	}
	// X-RateLimit-* headers should still be present on 429.
	if got := w.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Fatalf("expected X-RateLimit-Limit=2 on 429, got %q", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("expected X-RateLimit-Remaining=0 on 429, got %q", got)
	}
}

func TestRateLimit_RemainingDecrementsCorrectly(t *testing.T) {
	// burst=5, refill so slow it doesn't matter for the test window.
	router := newRateLimitTestRouter(t, rate.Every(time.Hour), 5)
	want := []string{"4", "3", "2", "1", "0"}
	for i, w := range want {
		resp := doRateLimitedReq(router)
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, resp.Code)
		}
		got := resp.Header().Get("X-RateLimit-Remaining")
		if got != w {
			t.Fatalf("request %d: expected X-RateLimit-Remaining=%s, got %s", i+1, w, got)
		}
	}
	// Sixth request must 429 with Remaining=0.
	resp := doRateLimitedReq(router)
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request: expected 429, got %d", resp.Code)
	}
	if got := resp.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("6th request: expected Remaining=0, got %q", got)
	}
}
