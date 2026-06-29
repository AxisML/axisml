package auth

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// RateLimiter is a per-key token-bucket throttle used to slow online password
// brute-force against /auth/login. It is in-memory (per replica): a determined
// distributed attacker is not fully stopped, but unauthenticated request rate
// per source is bounded without any external dependency. Buckets refill lazily
// on access and stale ones are pruned opportunistically, so no background
// goroutine is needed.
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	capacity   float64
	refillPerS float64
	now        func() time.Time
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing bursts up to capacity, refilling
// refillPerSecond tokens per second.
func NewRateLimiter(capacity int, refillPerSecond float64) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*bucket),
		capacity:   float64(capacity),
		refillPerS: refillPerSecond,
		now:        func() time.Time { return time.Now() },
	}
}

// allow reports whether the key may proceed, consuming one token when it can.
func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		// Opportunistic prune so the map can't grow without bound under a churn
		// of distinct keys (e.g. spoofed source addresses).
		if len(l.buckets) > 10000 {
			l.pruneLocked(now)
		}
		b = &bucket{tokens: l.capacity, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens += elapsed * l.refillPerS
		if b.tokens > l.capacity {
			b.tokens = l.capacity
		}
		b.lastSeen = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// pruneLocked drops buckets idle long enough to have fully refilled (they carry
// no state worth keeping). Caller must hold l.mu.
func (l *RateLimiter) pruneLocked(now time.Time) {
	var full time.Duration
	if l.refillPerS > 0 {
		full = time.Duration(l.capacity/l.refillPerS) * time.Second
	}
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > full {
			delete(l.buckets, k)
		}
	}
}

// Middleware returns a gin handler that rejects requests once the key (derived
// by keyFunc, e.g. client IP) exhausts its bucket, with a 429 problem response.
func (l *RateLimiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(keyFunc(c)) {
			fail(c, apperrors.New(apperrors.ClassTooManyReq,
				"too many requests; slow down").WithReason("rate-limited"))
			return
		}
		c.Next()
	}
}

// ClientIPKey keys the limiter by the caller's client IP.
func ClientIPKey(c *gin.Context) string { return c.ClientIP() }
