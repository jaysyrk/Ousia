package middleware

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const shardCount = 256

type shard struct {
	clients sync.Map
}

type rateLimiter struct {
	shards     [shardCount]*shard
	rate       int64
	burst      int64
	interval   time.Duration
	bucketPool sync.Pool
}

type bucket struct {
	tokens   int64
	lastSeen int64
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
		rate:     int64(requestsPerSecond),
		burst:    int64(burst),
		interval: time.Second,
		bucketPool: sync.Pool{
			New: func() interface{} {
				return &bucket{}
			},
		},
	}

	for i := 0; i < shardCount; i++ {
		rl.shards[i] = &shard{}
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

	now := time.Now().UnixNano()

	val, loaded := s.clients.Load(ip)
	if !loaded {
		b := rl.bucketPool.Get().(*bucket)
		atomic.StoreInt64(&b.tokens, rl.burst-1)
		atomic.StoreInt64(&b.lastSeen, now)

		s.clients.Store(ip, b)
		return true
	}

	b := val.(*bucket)
	lastSeen := atomic.LoadInt64(&b.lastSeen)
	elapsed := time.Duration(now - lastSeen)

	refill := int64(elapsed/rl.interval) * rl.rate
	if refill > 0 {
		if atomic.CompareAndSwapInt64(&b.lastSeen, lastSeen, now) {
			atomic.AddInt64(&b.tokens, refill)

			// Cap the tokens at the max burst
			currentTokens := atomic.LoadInt64(&b.tokens)
			if currentTokens > rl.burst {
				atomic.StoreInt64(&b.tokens, rl.burst)
			}
		}
	}

	// Lock-free decrement using CAS loop
	for {
		currentTokens := atomic.LoadInt64(&b.tokens)
		if currentTokens <= 0 {
			return false
		}
		if atomic.CompareAndSwapInt64(&b.tokens, currentTokens, currentTokens-1) {
			return true
		}
	}
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixNano()
		for i := 0; i < shardCount; i++ {
			s := rl.shards[i]
			s.clients.Range(func(key, value interface{}) bool {
				b := value.(*bucket)
				lastSeen := atomic.LoadInt64(&b.lastSeen)
				if time.Duration(now-lastSeen) > 10*time.Minute {
					s.clients.Delete(key)
					rl.bucketPool.Put(b)
				}
				return true
			})
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
