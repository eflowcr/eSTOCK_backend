package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newCORSTestRouter builds a minimal Gin router with the CORS middleware and
// a single GET /ping handler that returns 200 OK.
func newCORSTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "https://estock.eflowsuite.com,http://localhost:4200")
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.OPTIONS("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func TestCORS_ReflectsAllowedOrigin(t *testing.T) {
	r := newCORSTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://estock.eflowsuite.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://estock.eflowsuite.com" {
		t.Fatalf("expected Allow-Origin to reflect request Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Allow-Credentials=true, got %q", got)
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", got)
	}
}

func TestCORS_RejectsUnknownOrigin(t *testing.T) {
	r := newCORSTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for unknown origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected no Allow-Credentials for unknown origin, got %q", got)
	}
	// Non-preflight passes through to handler (browser will reject the response).
	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to run for non-preflight unknown origin, got status %d", w.Code)
	}
}

func TestCORS_AllowsCredentialsOnlyWithSpecificOrigin(t *testing.T) {
	r := newCORSTestRouter(t)
	// Sweep several origins (allowed + denied) and assert: when Allow-Credentials
	// is present, Allow-Origin MUST be a specific origin, never "*".
	cases := []struct {
		origin     string
		shouldPass bool
	}{
		{"https://estock.eflowsuite.com", true},
		{"http://localhost:4200", true},
		{"https://attacker.example.com", false},
		{"", false},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		r.ServeHTTP(w, req)

		ao := w.Header().Get("Access-Control-Allow-Origin")
		if ao == "*" {
			t.Fatalf("origin %q: Allow-Origin must NEVER be \"*\" when credentials are in play", c.origin)
		}
		creds := w.Header().Get("Access-Control-Allow-Credentials")
		if creds == "true" && ao == "" {
			t.Fatalf("origin %q: Allow-Credentials=true without Allow-Origin (spec violation)", c.origin)
		}
		if c.shouldPass {
			if ao != c.origin {
				t.Fatalf("origin %q: expected reflected, got %q", c.origin, ao)
			}
		} else {
			if ao != "" {
				t.Fatalf("origin %q: expected no Allow-Origin, got %q", c.origin, ao)
			}
		}
	}
}

func TestCORS_PreflightRejectsUnknownOrigin(t *testing.T) {
	r := newCORSTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown-origin preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin on rejected preflight, got %q", got)
	}
}

func TestCORS_PreflightAllowsKnownOrigin(t *testing.T) {
	r := newCORSTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://estock.eflowsuite.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for allowed-origin preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://estock.eflowsuite.com" {
		t.Fatalf("expected reflected Allow-Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatalf("expected Allow-Methods on preflight, got empty")
	}
}

func TestCORS_DefaultAllowlistWhenEnvUnset(t *testing.T) {
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "")
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://estock.eflowsuite.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://estock.eflowsuite.com" {
		t.Fatalf("expected default allowlist to permit prod origin, got %q", got)
	}
}
