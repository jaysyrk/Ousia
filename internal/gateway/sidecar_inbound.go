package gateway

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	upstreams map[string]string
}

func NewOutboundProxy(upstreams map[string]string) *OutboundProxy {
	return &OutboundProxy{upstreams: upstreams}
}

func (p *OutboundProxy) Start(listenAddr string) error {
	fmt.Printf("sidecar outbound: listening on %s\n", listenAddr)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go p.handle(conn)
	}
}

func (p *OutboundProxy) handle(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	target, ok := p.upstreams[string(buf[:n])]
	if !ok {
		return
	}

	upstream, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer upstream.Close()

	go io.Copy(upstream, conn)
	io.Copy(conn, upstream)
}
