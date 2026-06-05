package gateway

import (
	"errors"
	"net"
	"sync"
	"time"
)

type bouncerBucket struct {
	tokens   int
	lastSeen time.Time
}

type BouncerListener struct {
	net.Listener
	mu       sync.Mutex
	clients  map[string]*bouncerBucket
	rate     int
	burst    int
	interval time.Duration
}

func NewBouncerListener(inner net.Listener, rate, burst int) *BouncerListener {
	return &BouncerListener{
		Listener: inner,
		clients:  make(map[string]*bouncerBucket),
		rate:     rate,
		burst:    burst,
		interval: time.Second,
	}
}

func (b *BouncerListener) Accept() (net.Conn, error) {
	conn, err := b.Listener.Accept()
	if err != nil {
		return nil, err
	}

	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		host = conn.RemoteAddr().String()
	}

	if !b.allow(host) {
		conn.Close()
		return nil, errors.New("connection rate limit exceeded")
	}

	return conn, nil
}

func (b *BouncerListener) allow(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	bk, ok := b.clients[key]
	if !ok {
		if len(b.clients) >= 10000 {
			return false
		}
		b.clients[key] = &bouncerBucket{tokens: b.burst - 1, lastSeen: time.Now()}
		return true
	}

	elapsed := time.Since(bk.lastSeen)
	refill := int(elapsed/b.interval) * b.rate
	bk.tokens += refill
	if bk.tokens > b.burst {
		bk.tokens = b.burst
	}
	bk.lastSeen = time.Now()

	if bk.tokens <= 0 {
		return false
	}

	bk.tokens--
	return true
}
