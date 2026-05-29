package gateway

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/observability"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/types"
)

type Handler struct {
	router    *router.Router
	balancers map[string]balancer.Balancer
}

func NewHandler(r *router.Router, balancers map[string]balancer.Balancer) *Handler {
	return &Handler{
		router:    r,
		balancers: balancers,
	}
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

	wrapped := observability.Middleware(route.Action.UpstreamPool, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		forward(w, req, endpoint, route)
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

func forward(w http.ResponseWriter, req *http.Request, ep *types.Endpoint, route *types.Route) {
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
	if i := strings.LastIndex(req.RemoteAddr, ":"); i != -1 {
		return req.RemoteAddr[:i]
	}
	return req.RemoteAddr
}
