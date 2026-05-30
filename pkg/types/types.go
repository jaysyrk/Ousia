package types

import (
	"sync/atomic"
	"time"
)

type Endpoint struct {
	ID		string
	Address		string
	Weight		int
	Healthy		atomic.Bool
	Metadata	map[string]string
}

type UpstreamPool struct {
	Name		string
	Endpoints	[]*Endpoint
	Algorithm	LBAlgorithm
}

type LBAlgorithm string

const (
	AlgoRoundRobin	LBAlgorithm	= "round-robin"
	AlgoWRR		LBAlgorithm	= "wrr"
	AlgoLeastConn	LBAlgorithm	= "least-conn"
	AlgoIPHash	LBAlgorithm	= "ip-hash"
)

type RouteMatch struct {
	PathPrefix	string
	PathExact	string
	Methods		[]string
	Headers		map[string]string
}

type RouteAction struct {
	UpstreamPool      string
	StripPrefix       string
	AddHeaders        map[string]string // add/override request headers before forwarding
	RemoveHeaders     []string          // strip request headers before forwarding
	AddRespHeaders    map[string]string // add/override response headers before sending to client
	RemoveRespHeaders []string          // strip response headers before sending to client
	Timeout           time.Duration
	RetryCount        int
}

type Route struct {
	ID		string
	Priority	int
	Match		RouteMatch
	Action		RouteAction
}

type VirtualHost struct {
	Hostname	string
	Routes		[]*Route
	TLS		*TLSConfig
}

type TLSConfig struct {
	CertFile	string
	KeyFile		string
}

type Policy struct {
	RateLimit	*RateLimitPolicy
	CircuitBreaker	*CircuitBreakerPolicy
}

type RateLimitPolicy struct {
	RequestsPerSecond	int
	BurstSize		int
}

type CircuitBreakerPolicy struct {
	Threshold	int
	Timeout		time.Duration
}

type HealthStatus struct {
	EndpointID	string
	Healthy		bool
	CheckedAt	time.Time
	Error		string
}
