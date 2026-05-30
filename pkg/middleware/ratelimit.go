package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*bucket
	rate     int
	burst    int
	interval time.Duration
	keyFn    func(*http.Request) string
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

func RateLimit(ctx context.Context, requestsPerSecond int, burst int) Middleware {
	return RateLimitWithKey(ctx, requestsPerSecond, burst, ExtractIP)
}

func RateLimitWithKey(ctx context.Context, requestsPerSecond int, burst int, keyFn func(*http.Request) string) Middleware {
	rl := &rateLimiter{
		clients:  make(map[string]*bucket),
		rate:     requestsPerSecond,
		burst:    burst,
		interval: time.Second,
		keyFn:    keyFn,
	}

	go rl.cleanup(ctx)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rl.keyFn(r)
			if !rl.allow(key) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func HeaderKeyFunc(header string) func(*http.Request) string {
	return func(r *http.Request) string {
		if val := r.Header.Get(header); val != "" {
			return val
		}
		return ExtractIP(r)
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.clients[key]
	if !ok {
		if len(rl.clients) >= 10000 {
			return false
		}
		rl.clients[key] = &bucket{tokens: rl.burst - 1, lastSeen: time.Now()}
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

func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			for key, b := range rl.clients {
				if time.Since(b.lastSeen) > 10*time.Minute {
					delete(rl.clients, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func ExtractIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

