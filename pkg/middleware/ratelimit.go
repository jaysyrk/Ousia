package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

const shardCount = 256

type shard struct {
	mu      sync.Mutex
	clients map[string]*bucket
}

type rateLimiter struct {
	shards   [shardCount]*shard
	rate     int
	burst    int
	interval time.Duration
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

func hashIP(ip string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(ip); i++ {
		hash ^= uint32(ip[i])
		hash *= 16777619
	}
	return hash % shardCount
}

func RateLimit(requestsPerSecond int, burst int) Middleware {
	rl := &rateLimiter{
		rate:     requestsPerSecond,
		burst:    burst,
		interval: time.Second,
	}

	for i := 0; i < shardCount; i++ {
		rl.shards[i] = &shard{
			clients: make(map[string]*bucket),
		}
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
	idx := hashIP(ip)
	s := rl.shards[idx]

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.clients[ip]
	if !ok {
		s.clients[ip] = &bucket{tokens: rl.burst - 1, lastSeen: time.Now()}
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
		for i := 0; i < shardCount; i++ {
			s := rl.shards[i]
			s.mu.Lock()
			for ip, b := range s.clients {
				if time.Since(b.lastSeen) > 10*time.Minute {
					delete(s.clients, ip)
				}
			}
			s.mu.Unlock()
		}
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
