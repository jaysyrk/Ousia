package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/jaysyrk/ousia/pkg/types"
)

type Router struct {
	mu    sync.RWMutex
	hosts map[string]*types.VirtualHost
}

func New(hosts []*types.VirtualHost) *Router {
	r := &Router{
		hosts: make(map[string]*types.VirtualHost, len(hosts)),
	}
	for _, vh := range hosts {
		sort.Slice(vh.Routes, func(i, j int) bool {
			return vh.Routes[i].Priority < vh.Routes[j].Priority
		})
		r.hosts[vh.Hostname] = vh
	}
	return r
}

func (r *Router) Match(req *http.Request) (*types.Route, *types.VirtualHost, error) {
	host := normalizeHost(req.Host)

	r.mu.RLock()
	vh, ok := r.hosts[host]
	if !ok {
		vh, ok = r.hosts["*"]
	}
	r.mu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("router: no virtual host for %q", host)
	}

	for _, route := range vh.Routes {
		if matchRoute(route, req) {
			return route, vh, nil
		}
	}

	return nil, nil, fmt.Errorf("router: no route matched %s %s", req.Method, req.URL.Path)
}

func (r *Router) AddVirtualHost(vh *types.VirtualHost) {
	sort.Slice(vh.Routes, func(i, j int) bool {
		return vh.Routes[i].Priority < vh.Routes[j].Priority
	})
	r.mu.Lock()
	r.hosts[vh.Hostname] = vh
	r.mu.Unlock()
}

func (r *Router) RemoveVirtualHost(hostname string) {
	r.mu.Lock()
	delete(r.hosts, hostname)
	r.mu.Unlock()
}

func matchRoute(route *types.Route, req *http.Request) bool {
	m := route.Match

	if len(m.Methods) > 0 && !containsString(m.Methods, req.Method) {
		return false
	}

	if m.PathExact != "" {
		if req.URL.Path != m.PathExact {
			return false
		}
	} else if m.PathPrefix != "" {
		if !strings.HasPrefix(req.URL.Path, m.PathPrefix) {
			return false
		}
	}

	for key, val := range m.Headers {
		if req.Header.Get(key) != val {
			return false
		}
	}

	return true
}

func normalizeHost(host string) string {
	if i := strings.LastIndex(host, ":"); i != -1 {
		return host[:i]
	}
	return host
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
