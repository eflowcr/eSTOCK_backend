package tools

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter holds per-IP token-bucket limiters.
// Stale entries (not seen in > 5 minutes) are cleaned up lazily.
type IPRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipLimiter
	r       rate.Limit
	b       int
}

func newIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	rl := &IPRateLimiter{
		entries: make(map[string]*ipLimiter),
		r:       r,
		b:       b,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *IPRateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, ok := rl.entries[ip]
	if !ok {
		e = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

// cleanupLoop removes IPs not seen in the last 5 minutes, preventing unbounded memory growth.
func (rl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, e := range rl.entries {
			if time.Since(e.lastSeen) > 5*time.Minute {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// NewIPRateLimiter returns a Gin middleware that limits each client IP to r requests
// per second with a burst of b. Excess requests receive HTTP 429.
//
// Example — 5 requests per minute with burst of 5:
//
//	tools.NewIPRateLimiter(rate.Every(time.Minute/5), 5)
func NewIPRateLimiter(r rate.Limit, b int) gin.HandlerFunc {
	rl := newIPRateLimiter(r, b)
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		if !rl.get(ip).Allow() {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "too_many_requests",
				"message": "Rate limit exceeded. Please wait before retrying.",
			})
			return
		}
		ctx.Next()
	}
}
