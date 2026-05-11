package tools

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
)

// idempotencyKeyHeader is the HTTP header mobile sends per mutation. Value
// is a client-generated UUID v4. The same value is sent on every retry so
// the backend can dedupe.
const idempotencyKeyHeader = "Idempotency-Key"

// idempotencyTTL is the window during which a cached response is replayed.
// After this, a replay re-applies the mutation — operators shouldn't be
// queuing offline writes for longer than a week, and the cron sweep frees
// the row.
const idempotencyTTL = 7 * 24 * time.Hour

// Cache only response status codes that semantically indicate a *completed*
// mutation. Caching 5xx would defeat the point — operator retries to RECOVER
// from a transient backend failure. Caching client errors (4xx) is correct
// because the same key replayed would produce the same validation rejection.
func isCacheableStatus(status int) bool {
	return status >= 200 && status < 500
}

// IdempotencyMiddleware returns a Gin middleware that dedupes mobile mutations
// by `Idempotency-Key` header. Mount it AFTER JWTAuthMiddleware (needs user_id
// on context) and AFTER RequirePermission (don't want to cache 403s).
//
// Behavior:
//  1. No header → pass-through (legacy mobile clients + non-mobile callers).
//  2. Header + cache hit + not expired → return cached status + body; the
//     downstream handler does NOT run.
//  3. Header + cache miss → run handler, capture response, store if cacheable.
//
// Repo may be nil for tests, in which case the middleware short-circuits to
// Next() (same pattern as RequirePermission).
func IdempotencyMiddleware(repo ports.IdempotencyKeysRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if repo == nil {
			c.Next()
			return
		}

		key := c.GetHeader(idempotencyKeyHeader)
		if key == "" {
			c.Next()
			return
		}

		userID := c.GetString(ContextKeyUserID)
		if userID == "" {
			// No user context = before JWT middleware. The route is misconfigured
			// (idempotency relies on user scoping) but don't 500 the operator;
			// just pass through and let the handler decide.
			c.Next()
			return
		}

		// Hit path: return cached response and short-circuit.
		ctx := c.Request.Context()
		cached, err := repo.Lookup(ctx, key, userID)
		if err == nil && cached != nil {
			c.Header("X-Idempotent-Replayed", "true")
			c.Data(cached.ResponseStatus, "application/json; charset=utf-8", cached.ResponseBody)
			c.Abort()
			return
		}
		// Lookup error → log via gin and proceed without dedup (better to
		// re-apply than to fail the operator's request).
		if err != nil {
			c.Error(err)
		}

		// Miss path: wrap the writer so we can capture the response body, run
		// the handler chain, then persist the response under the key.
		writer := &responseCaptureWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		if c.IsAborted() {
			// The handler aborted (auth, permission, validation). Still cache
			// the response if the status is in the cacheable range — replay
			// will surface the same 4xx, which is the desired idempotent
			// behavior (the same input produces the same rejection).
		}

		if !isCacheableStatus(writer.Status()) {
			return
		}

		// Best-effort store: never bubble a DB error to the operator after
		// the handler succeeded. Logged via gin.Context for observability.
		entry := &database.IdempotencyKey{
			Key:            key,
			UserID:         userID,
			Endpoint:       c.FullPath(),
			Method:         c.Request.Method,
			ResponseStatus: writer.Status(),
			ResponseBody:   writer.buf.Bytes(),
			ExpiresAt:      time.Now().Add(idempotencyTTL),
		}
		if storeErr := repo.Store(context.Background(), entry); storeErr != nil {
			c.Error(storeErr)
		}
	}
}

// responseCaptureWriter buffers the response body so the middleware can
// persist it after the handler completes. Delegates everything else to the
// underlying Gin writer.
type responseCaptureWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *responseCaptureWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCaptureWriter) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// http.ResponseWriter sanity — gin.ResponseWriter already embeds it but the
// compiler needs an explicit signature when we override Write/WriteString.
var _ http.ResponseWriter = (*responseCaptureWriter)(nil)
