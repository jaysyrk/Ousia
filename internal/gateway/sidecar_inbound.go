package gateway

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type InboundProxy struct {
	localPort int
}

func NewInboundProxy(localPort int) *InboundProxy {
	return &InboundProxy{localPort: localPort}
}

func (p *InboundProxy) Start(listenAddr string) error {
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", p.localPort),
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "local service unavailable: "+err.Error(), http.StatusBadGateway)
	}

	fmt.Printf("sidecar inbound: listening on %s → local :%d\n", listenAddr, p.localPort)
	return http.ListenAndServe(listenAddr, proxy)
}

type OutboundProxy struct {
	mapper *ServiceMapper
}

func NewOutboundProxy(mapper *ServiceMapper) *OutboundProxy {
	return &OutboundProxy{mapper: mapper}
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
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, req)
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
