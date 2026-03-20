package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// visitorCleanupInterval is how often stale rate limiter entries are purged.
	visitorCleanupInterval = time.Minute
	// visitorStaleThreshold is how long a visitor must be inactive before removal.
	visitorStaleThreshold = 3 * time.Minute
	// staticPathPrefix is the URL prefix for static assets (skipped by rate limiter).
	staticPathPrefix = "/static/"
	// maxVisitors is the upper bound on tracked IPs to prevent memory exhaustion.
	maxVisitors = 10000
)

// RateLimiter provides per-IP rate limiting for HTTP requests.
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
	stop     chan struct{}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter that allows r requests per second
// with a burst capacity of b.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    b,
		stop:     make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl
}

// getVisitor retrieves or creates a rate limiter for the given IP.
// Returns nil if the visitor map is at capacity and the IP is not already tracked.
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	if len(rl.visitors) >= maxVisitors {
		return nil
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

// cleanupLoop removes stale visitor entries every minute.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(visitorCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > visitorStaleThreshold {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

// Middleware returns HTTP middleware that rate-limits requests per IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health checks and static assets
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		if len(r.URL.Path) >= len(staticPathPrefix) && r.URL.Path[:len(staticPathPrefix)] == staticPathPrefix {
			next.ServeHTTP(w, r)
			return
		}

		ip := extractIP(r)
		limiter := rl.getVisitor(ip)

		if limiter == nil {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractIP gets the client IP from the request, handling proxied requests.
func extractIP(r *http.Request) string {
	// Use RemoteAddr directly (don't trust X-Forwarded-For from untrusted sources)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
