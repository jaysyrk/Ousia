package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu		sync.Mutex
	clients		map[string]*bucket
	rate		int
	burst		int
	interval	time.Duration
}

type bucket struct {
	tokens		int
	lastSeen	time.Time
}

func RateLimit(requestsPerSecond int, burst int) Middleware {
	rl := &rateLimiter{
		clients:	make(map[string]*bucket),
		rate:		requestsPerSecond,
		burst:		burst,
		interval:	time.Second,
	}

	go rl.cleanup()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !rl.allow(ip) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.clients[ip]
	if !ok {
		rl.clients[ip] = &bucket{tokens: rl.burst - 1, lastSeen: time.Now()}
		return true
	}

	elapsed := time.Since(b.lastSeen)
	refill := int(elapsed/rl.interval) * rl.rate
	b.tokens += refill
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastSeen = time.Now()

	if b.tokens <= 0 {
		return false
	}

	b.tokens--
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, b := range rl.clients {
			if time.Since(b.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	if i := strings.LastIndex(r.RemoteAddr, ":"); i != -1 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}
