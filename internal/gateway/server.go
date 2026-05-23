package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jaysyrk/ousia/pkg/middleware"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, handler http.Handler) *Server {
	chain := middleware.Chain(
		handler,
		middleware.RequestID,
		middleware.RateLimit(100, 200),
		middleware.CORS(middleware.DefaultCORSConfig()),
		middleware.Timeout(30*time.Second),
	)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      chain,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	fmt.Printf("Ousia Gateway listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
