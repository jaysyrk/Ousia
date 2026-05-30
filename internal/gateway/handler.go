package gateway

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/middleware"
	"github.com/jaysyrk/ousia/internal/observability"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/types"
)

type Handler struct {
	router    *router.Router
	balancers map[string]balancer.Balancer
	cbMu      sync.RWMutex
	cbs       map[string]*middleware.CircuitBreaker
}

func NewHandler(r *router.Router, balancers map[string]balancer.Balancer) *Handler {
	return &Handler{
		router:    r,
		balancers: balancers,
		cbs:       make(map[string]*middleware.CircuitBreaker),
	}
}

func (h *Handler) getCircuitBreaker(ep *types.Endpoint) *middleware.CircuitBreaker {
	h.cbMu.RLock()
	cb, ok := h.cbs[ep.ID]
	h.cbMu.RUnlock()

	if ok {
		return cb
	}

	h.cbMu.Lock()
	defer h.cbMu.Unlock()

	if cb, ok := h.cbs[ep.ID]; ok {
		return cb
	}

	cb = middleware.NewCircuitBreaker(3, 5*time.Second)
	h.cbs[ep.ID] = cb
	return cb
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, _, err := h.router.Match(req)
	if err != nil {
		http.Error(w, "no route matched", http.StatusNotFound)
		return
	}

	lb, ok := h.balancers[route.Action.UpstreamPool]
	if !ok {
		http.Error(w, "upstream pool not found", http.StatusBadGateway)
		return
	}

	endpoint, err := lb.Next(clientIP(req))
	if err != nil {
		http.Error(w, "no healthy upstream", http.StatusServiceUnavailable)
		return
	}
	defer lb.Done(endpoint.ID)

	for key, val := range route.Action.AddHeaders {
		req.Header.Set(key, val)
	}
	for _, key := range route.Action.RemoveHeaders {
		req.Header.Del(key)
	}

	rw := &respHeaderWriter{
		ResponseWriter: w,
		addHeaders:     route.Action.AddRespHeaders,
		removeHeaders:  route.Action.RemoveRespHeaders,
	}

	cb := h.getCircuitBreaker(endpoint)

	wrapped := observability.Middleware(route.Action.UpstreamPool, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		forward(w, req, endpoint, route, cb)
	}))

	wrapped.ServeHTTP(rw, req)
}

type respHeaderWriter struct {
	http.ResponseWriter
	addHeaders     map[string]string
	removeHeaders  []string
	headersMutated bool
}

func (rw *respHeaderWriter) WriteHeader(code int) {
	if !rw.headersMutated {
		rw.headersMutated = true
		for key, val := range rw.addHeaders {
			rw.ResponseWriter.Header().Set(key, val)
		}
		for _, key := range rw.removeHeaders {
			rw.ResponseWriter.Header().Del(key)
		}
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *respHeaderWriter) Write(b []byte) (int, error) {
	if !rw.headersMutated {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

type resiliencyTransport struct {
	base  http.RoundTripper
	route *types.Route
	cb    *middleware.CircuitBreaker
}

func (t *resiliencyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	retryCfg := middleware.DefaultRetryConfig()
	if t.route.Action.RetryCount > 0 {
		retryCfg.MaxAttempts = t.route.Action.RetryCount + 1
	}

	err := middleware.WithRetry(retryCfg, func() error {
		if t.cb != nil {
			if err := t.cb.Allow(); err != nil {
				return err
			}
		}

		var err error
		resp, err = t.base.RoundTrip(req)

		if err != nil {
			if t.cb != nil {
				t.cb.Failure()
			}
			return err
		}

		if resp.StatusCode >= 500 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if t.cb != nil {
				t.cb.Failure()
			}
			return fmt.Errorf("upstream returned %d", resp.StatusCode)
		}

		if t.cb != nil {
			t.cb.Success()
		}
		return nil
	})

	if err != nil && resp == nil {
		return nil, err
	}
	return resp, nil
}

func forward(w http.ResponseWriter, req *http.Request, ep *types.Endpoint, route *types.Route, cb *middleware.CircuitBreaker) {
	if route.Action.Timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), route.Action.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	target := &url.URL{
		Scheme: "http",
		Host:   ep.Address,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &resiliencyTransport{
		base:  http.DefaultTransport,
		route: route,
		cb:    cb,
	}

	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host

		if route.Action.StripPrefix != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Action.StripPrefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		r.Header.Set("X-Forwarded-Host", req.Host)
		r.Header.Set("X-Origin-Host", target.Host)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, req)
}

func clientIP(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
