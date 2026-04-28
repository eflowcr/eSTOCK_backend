package tools

import (
	"math"
	"net/http"
	"strconv"
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

// secondsPerToken returns the time (seconds) it takes to refill one token.
// Returns 0 if the rate is non-positive (degenerate config).
func (rl *IPRateLimiter) secondsPerToken() float64 {
	if rl.r <= 0 {
		return 0
	}
	return 1.0 / float64(rl.r)
}

// computeHeaders derives the X-RateLimit-* values for a given limiter snapshot.
// Returns (remaining, resetUnix, retryAfterSeconds). retryAfterSeconds is >= 1
// when remaining == 0, otherwise 0 (caller decides whether to emit it).
func (rl *IPRateLimiter) computeHeaders(lim *rate.Limiter, now time.Time) (int, int64, int) {
	tokens := lim.TokensAt(now)
	if tokens < 0 {
		tokens = 0
	}
	remaining := int(math.Floor(tokens))
	if remaining > rl.b {
		remaining = rl.b
	}

	spt := rl.secondsPerToken()
	// Reset = time when bucket is fully refilled to burst capacity.
	missing := float64(rl.b) - tokens
	if missing < 0 {
		missing = 0
	}
	resetSeconds := missing * spt
	resetAt := now.Add(time.Duration(resetSeconds * float64(time.Second))).Unix()

	retryAfter := 0
	if remaining == 0 {
		// Time until next single token is available.
		secsUntilNext := (1.0 - tokens) * spt
		if secsUntilNext < 1 {
			secsUntilNext = 1
		}
		retryAfter = int(math.Ceil(secsUntilNext))
	}
	return remaining, resetAt, retryAfter
}

// NewIPRateLimiter returns a Gin middleware that limits each client IP to r requests
// per second with a burst of b. Excess requests receive HTTP 429.
//
// Every response (allowed or 429) carries:
//   - X-RateLimit-Limit:     burst capacity
//   - X-RateLimit-Remaining: tokens left after this request
//   - X-RateLimit-Reset:     unix timestamp when bucket is fully refilled
//
// On 429 only, the response also carries:
//   - Retry-After: seconds until the next token is available (RFC 6585, integer)
//
// Example — 5 requests per minute with burst of 5:
//
//	tools.NewIPRateLimiter(rate.Every(time.Minute/5), 5)
func NewIPRateLimiter(r rate.Limit, b int) gin.HandlerFunc {
	rl := newIPRateLimiter(r, b)
	limitStr := strconv.Itoa(b)
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		lim := rl.get(ip)
		now := time.Now()
		allowed := lim.AllowN(now, 1)

		remaining, resetAt, retryAfter := rl.computeHeaders(lim, now)

		ctx.Header("X-RateLimit-Limit", limitStr)
		ctx.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		ctx.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if !allowed {
			if retryAfter < 1 {
				retryAfter = 1
			}
			ctx.Header("Retry-After", strconv.Itoa(retryAfter))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "too_many_requests",
				"message": "Rate limit exceeded. Please wait before retrying.",
			})
			return
		}
		ctx.Next()
	}
}
