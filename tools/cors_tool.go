package tools

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// defaultAllowedOrigins is the conservative allowlist used when the
// ALLOWED_ORIGINS env var is empty. Covers prod/qa/dev hosted envs and
// common local dev servers (Angular CLI 4200, Vite 5173).
var defaultAllowedOrigins = []string{
	"https://estock.eflowsuite.com",
	"https://estock-qa.eflowsuite.com",
	"https://estock-dev.eflowsuite.com",
	"http://localhost:4200",
	"http://localhost:5173",
}

var (
	allowedOriginsOnce sync.Once
	allowedOriginsSet  map[string]struct{}
)

// loadAllowedOrigins reads ALLOWED_ORIGINS once (comma-separated). Trims spaces
// and skips empty entries. Falls back to defaultAllowedOrigins when unset/empty.
func loadAllowedOrigins() map[string]struct{} {
	allowedOriginsOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
		set := make(map[string]struct{})
		if raw == "" {
			for _, o := range defaultAllowedOrigins {
				set[o] = struct{}{}
			}
		} else {
			for _, part := range strings.Split(raw, ",") {
				origin := strings.TrimSpace(part)
				if origin != "" {
					set[origin] = struct{}{}
				}
			}
		}
		allowedOriginsSet = set
	})
	return allowedOriginsSet
}

// resetAllowedOriginsForTest is a test-only hook to reset the once-initialized
// allowlist. It MUST NOT be called outside *_test.go files.
func resetAllowedOriginsForTest() {
	allowedOriginsOnce = sync.Once{}
	allowedOriginsSet = nil
}

// isOriginAllowed returns true when the request Origin is in the allowlist.
func isOriginAllowed(origin string) bool {
	return IsAllowedOrigin(origin)
}

// IsAllowedOrigin reports whether origin is present in the ALLOWED_ORIGINS
// allowlist (or the default allowlist when the env var is unset/empty).
// It is exported so other packages (e.g. tools.ResolveFrontendURL) can reuse
// the same allowlist without duplicating parsing logic.
func IsAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	_, ok := loadAllowedOrigins()[origin]
	return ok
}

// CORSMiddleware enforces CORS with an explicit allowlist. It reflects the
// request Origin header back when allowed (never `*`) so it can be safely
// combined with Allow-Credentials: true (browsers reject `*` + credentials).
//
// Allowlist source: ALLOWED_ORIGINS env var (comma-separated). When unset,
// uses the safe default list (prod/qa/dev/localhost).
//
// Behavior:
//   - Allowed Origin: reflects Origin in Allow-Origin, sets Allow-Credentials,
//     Allow-Headers, Allow-Methods, and Vary: Origin.
//   - Disallowed/missing Origin on non-preflight: passes through with no CORS
//     headers (browser will block the response, which is the correct CORS reject).
//   - OPTIONS preflight from disallowed Origin: 403.
//   - OPTIONS preflight from allowed Origin: 204 with full CORS headers.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := isOriginAllowed(origin)

		// Vary: Origin so caches don't serve a CORS response for the wrong origin.
		c.Header("Vary", "Origin")

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "POST, HEAD, PATCH, OPTIONS, GET, PUT, DELETE")
		}

		if c.Request.Method == http.MethodOptions {
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
