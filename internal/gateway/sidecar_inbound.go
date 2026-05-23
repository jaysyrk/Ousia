package gateway

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaysyrk/ousia/internal/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type statusRecorder struct {
	http.ResponseWriter
	status	int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

type InboundProxy struct {
	localPort	int
	serviceID	string
}

func NewInboundProxy(localPort int, serviceID string) *InboundProxy {
	return &InboundProxy{localPort: localPort, serviceID: serviceID}
}

func (p *InboundProxy) Start(listenAddr string) error {
	target := &url.URL{
		Scheme:	"http",
		Host:	fmt.Sprintf("127.0.0.1:%d", p.localPort),
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "local service unavailable: "+err.Error(), http.StatusBadGateway)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/stats" {
			promhttp.Handler().ServeHTTP(w, req)
			return
		}

		start := time.Now()
		traceID := req.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
			req.Header.Set("X-Trace-Id", traceID)
		}

		source := req.Header.Get("X-Ousia-Source")
		if source == "" {
			source = "unknown"
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		proxy.ServeHTTP(rec, req)

		durationMs := float64(time.Since(start).Milliseconds())
		status := fmt.Sprintf("%d", rec.status)

		observability.MeshRequestsTotal.WithLabelValues(source, p.serviceID, status, req.Method).Inc()
		observability.MeshRequestDuration.WithLabelValues(source, p.serviceID, req.Method).Observe(durationMs)
		observability.RequestLog(traceID, req.Method, req.URL.Path, req.Host, "local:"+fmt.Sprint(p.localPort), rec.status, durationMs)
	})

	fmt.Printf("sidecar inbound: listening on %s → local :%d\n", listenAddr, p.localPort)
	return http.ListenAndServe(listenAddr, handler)
}

type OutboundProxy struct {
	mapper		*ServiceMapper
	serviceID	string
}

func NewOutboundProxy(mapper *ServiceMapper, serviceID string) *OutboundProxy {
	return &OutboundProxy{mapper: mapper, serviceID: serviceID}
}

func (p *OutboundProxy) Start(listenAddr string) error {
	server := &http.Server{Addr: listenAddr, Handler: p}
	fmt.Printf("sidecar outbound: listening on %s\n", listenAddr)
	return server.ListenAndServe()
}

func (p *OutboundProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	serviceName := serviceNameFromRequest(req)
	targetHost, ok := p.mapper.Next(serviceName)
	if !ok {
		http.Error(w, "service not available", http.StatusBadGateway)
		return
	}

	target := &url.URL{Scheme: "http", Host: targetHost}
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host
		r.Header.Set("X-Forwarded-Host", req.Host)
		r.Header.Set("X-Origin-Host", target.Host)

		if r.Header.Get("X-Trace-Id") == "" {
			r.Header.Set("X-Trace-Id", uuid.New().String())
		}

		r.Header.Set("X-Ousia-Source", p.serviceID)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}

	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

	proxy.ServeHTTP(rec, req)

	durationMs := float64(time.Since(start).Milliseconds())
	status := fmt.Sprintf("%d", rec.status)

	observability.MeshRequestsTotal.WithLabelValues(p.serviceID, serviceName, status, req.Method).Inc()
	observability.MeshRequestDuration.WithLabelValues(p.serviceID, serviceName, req.Method).Observe(durationMs)
}

func serviceNameFromRequest(req *http.Request) string {
	if service := req.Header.Get("X-Ousia-Service"); service != "" {
		return service
	}
	host := req.Host
	if i := strings.IndexByte(host, ':'); i != -1 {
		host = host[:i]
	}
	return host
}
