package tools

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	requestIDHeader     = "X-Request-ID"
	correlationIDHeader  = "X-Correlation-ID"
	requestIDContextKey = "request_id"
)

// RequestLogMiddleware assigns a request/correlation ID (from header or generated), sets it on context
// and response header, then logs each request with method, path, status, duration, client IP, and request_id.
// Does not log request body or headers (they may contain secrets).
// Register after Recovery and CORS so panics are recovered first.
func RequestLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = c.GetHeader(correlationIDHeader)
		}
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set(requestIDContextKey, requestID)
		c.Header(requestIDHeader, requestID)

		start := time.Now()
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Info().
			Str("request_id", requestID).
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", clientIP).
			Msg("request")
	}
}

// GetRequestID returns the request/correlation ID from the context, or empty string if not set.
// Handlers can use this when logging or passing to downstream services.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(requestIDContextKey); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
