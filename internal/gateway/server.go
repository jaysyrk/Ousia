package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
	"github.com/jaysyrk/ousia/pkg/middleware"
)

type Server struct {
	httpServer *http.Server
	tlsServer  *http.Server
	tlsAddr    string
	cancel     context.CancelFunc
}

func NewServer(cfg *config.OusiaConfig, r *router.Router, balancers map[string]balancer.Balancer, handler http.Handler, tlsCfg *tls.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	vhostLimiters := buildVhostLimiters(ctx, cfg)
	rateLimited := applyVhostRateLimiting(handler, vhostLimiters)

	chain := middleware.Chain(
		rateLimited,
		middleware.RequestID,
		middleware.CORS(middleware.DefaultCORSConfig()),
		middleware.Timeout(30*time.Second),
	)

	httpHandler := chain
	if cfg.Gateway.TLSAddr != "" {
		httpHandler = redirectToHTTPS(cfg.Gateway.TLSAddr)
	}

	s := &Server{
		tlsAddr: cfg.Gateway.TLSAddr,
		httpServer: &http.Server{
			Addr:         cfg.Gateway.ListenAddr,
			Handler:      httpHandler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		cancel: cancel,
	}

	if tlsCfg != nil && cfg.Gateway.TLSAddr != "" {
		s.tlsServer = newTLSServer(cfg.Gateway.TLSAddr, chain, tlsCfg)
	}

	return s
}

func buildVhostLimiters(ctx context.Context, cfg *config.OusiaConfig) map[string]middleware.Middleware {
	limiters := make(map[string]middleware.Middleware)
	for _, vh := range cfg.VirtualHosts {
		if vh.RateLimit == nil {
			continue
		}
		rl := vh.RateLimit
		rps := rl.RequestsPerSecond
		burst := rl.Burst
		if burst <= 0 {
			burst = rps * 2
		}

		keyFn := middleware.ExtractIP
		if strings.HasPrefix(rl.KeyBy, "header:") {
			headerName := strings.TrimPrefix(rl.KeyBy, "header:")
			keyFn = middleware.HeaderKeyFunc(headerName)
		}

		limiters[vh.Hostname] = middleware.RateLimitWithKey(ctx, rps, burst, keyFn)
	}
	return limiters
}

func applyVhostRateLimiting(next http.Handler, limiters map[string]middleware.Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if i := strings.LastIndex(host, ":"); i != -1 {
			host = host[:i]
		}
		if limiter, ok := limiters[host]; ok {
			limiter(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	fmt.Printf("Ousia Gateway listening on %s\n", s.httpServer.Addr)
	if s.tlsServer != nil {
		go func() {
			fmt.Printf("Ousia Gateway TLS listening on %s\n", s.tlsAddr)
			if err := s.tlsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				fmt.Printf("TLS server error: %v\n", err)
			}
		}()
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	var tlsErr error
	if s.tlsServer != nil {
		tlsErr = s.tlsServer.Shutdown(ctx)
	}
	httpErr := s.httpServer.Shutdown(ctx)
	return errors.Join(tlsErr, httpErr)
}

