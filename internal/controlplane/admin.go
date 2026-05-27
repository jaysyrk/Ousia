package controlplane

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jaysyrk/ousia/internal/balancer"
	"github.com/jaysyrk/ousia/internal/observability"
	"github.com/jaysyrk/ousia/internal/router"
	"github.com/jaysyrk/ousia/pkg/config"
	"github.com/jaysyrk/ousia/pkg/types"
)

type AdminAPI struct {
	router		*router.Router
	balancers	map[string]balancer.Balancer
	store		*Store
	mesh		*MeshRegistry
}

func NewAdminAPI(r *router.Router, balancers map[string]balancer.Balancer, store *Store, mesh *MeshRegistry) *AdminAPI {
	return &AdminAPI{
		router:		r,
		balancers:	balancers,
		store:		store,
		mesh:		mesh,
	}
}

func (a *AdminAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/routes", a.handleRoutes)
	mux.HandleFunc("/api/routes/", a.handleRouteByID)
	mux.HandleFunc("/api/upstreams", a.handleUpstreams)
	mux.HandleFunc("/api/upstreams/", a.handleUpstreamEndpoints)
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/stats", a.handleStats)

	mux.HandleFunc("/api/mesh/register", a.handleMeshRegister)
	mux.HandleFunc("/api/mesh/heartbeat", a.handleMeshHeartbeat)
	mux.HandleFunc("/api/mesh/deregister", a.handleMeshDeregister)
	mux.HandleFunc("/api/mesh/services", a.handleMeshServices)
}

func (a *AdminAPI) handleRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listRoutes(w, r)
	case http.MethodPost:
		a.addRoute(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminAPI) handleRouteByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/routes/")
	if id == "" {
		http.Error(w, "route id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		a.getRoute(w, r, id)
	case http.MethodPut:
		a.updateRoute(w, r, id)
	case http.MethodDelete:
		a.deleteRoute(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminAPI) handleUpstreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.listUpstreams(w, r)
}

func (a *AdminAPI) handleUpstreamEndpoints(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/upstreams/"), "/")
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	poolName := parts[0]
	if len(parts) == 2 && parts[1] == "endpoints" {
		if r.Method == http.MethodPost {
			a.addEndpoint(w, r, poolName)
			return
		}
	}
	if len(parts) == 3 && parts[1] == "endpoints" {
		if r.Method == http.MethodDelete {
			a.removeEndpoint(w, r, poolName, parts[2])
			return
		}
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (a *AdminAPI) listRoutes(w http.ResponseWriter, r *http.Request) {
	cfg := a.store.Get()
	type routeResponse struct {
		ID		string			`json:"id"`
		VirtualHost	string			`json:"virtual_host"`
		Priority	int			`json:"priority"`
		PathPrefix	string			`json:"path_prefix,omitempty"`
		PathExact	string			`json:"path_exact,omitempty"`
		Methods		[]string		`json:"methods,omitempty"`
		Headers		map[string]string	`json:"headers,omitempty"`
		Upstream	string			`json:"upstream"`
		Timeout		string			`json:"timeout,omitempty"`
		RetryCount	int			`json:"retry_count,omitempty"`
	}

	var routes []routeResponse
	for _, vh := range cfg.VirtualHosts {
		for _, route := range vh.Routes {
			routes = append(routes, routeResponse{
				ID:		route.ID,
				VirtualHost:	vh.Hostname,
				Priority:	route.Priority,
				PathPrefix:	route.Match.PathPrefix,
				PathExact:	route.Match.PathExact,
				Methods:	route.Match.Methods,
				Headers:	route.Match.Headers,
				Upstream:	route.Action.Upstream,
				Timeout:	route.Action.Timeout,
				RetryCount:	route.Action.RetryCount,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"routes": routes, "count": len(routes)})
}

func (a *AdminAPI) getRoute(w http.ResponseWriter, r *http.Request, id string) {
	cfg := a.store.Get()
	vhIndex, _, route := findRouteIndexes(cfg, id)
	if route == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}
	vh := cfg.VirtualHosts[vhIndex]
	writeJSON(w, http.StatusOK, map[string]any{
		"id":		route.ID,
		"virtual_host":	vh.Hostname,
		"priority":	route.Priority,
		"path_prefix":	route.Match.PathPrefix,
		"path_exact":	route.Match.PathExact,
		"methods":	route.Match.Methods,
		"headers":	route.Match.Headers,
		"upstream":	route.Action.Upstream,
		"timeout":	route.Action.Timeout,
		"retry_count":	route.Action.RetryCount,
	})
}

func (a *AdminAPI) addRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		VirtualHost	string			`json:"virtual_host"`
		ID		string			`json:"id"`
		Priority	int			`json:"priority"`
		PathPrefix	string			`json:"path_prefix"`
		PathExact	string			`json:"path_exact"`
		Methods		[]string		`json:"methods"`
		Headers		map[string]string	`json:"headers"`
		Upstream	string			`json:"upstream"`
		Timeout		string			`json:"timeout"`
		RetryCount	int			`json:"retry_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.ID == "" || body.VirtualHost == "" || body.Upstream == "" {
		http.Error(w, "id, virtual_host, and upstream are required", http.StatusBadRequest)
		return
	}

	timeout, err := parseTimeout(body.Timeout)
	if err != nil {
		http.Error(w, "invalid timeout: "+err.Error(), http.StatusBadRequest)
		return
	}

	route := &types.Route{
		ID:		body.ID,
		Priority:	body.Priority,
		Match: types.RouteMatch{
			PathPrefix:	body.PathPrefix,
			PathExact:	body.PathExact,
			Methods:	body.Methods,
			Headers:	body.Headers,
		},
		Action: types.RouteAction{
			UpstreamPool:	body.Upstream,
			Timeout:	timeout,
			RetryCount:	body.RetryCount,
		},
	}

	cfg := cloneConfig(a.store.Get())
	addOrUpdateRouteInConfig(cfg, route, body.VirtualHost)
	a.store.Set(cfg)
	applyRouteToRouter(a.router, cfg, route, body.VirtualHost)

	writeJSON(w, http.StatusCreated, map[string]any{"message": "route added", "id": body.ID})
}

func (a *AdminAPI) updateRoute(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		VirtualHost	string			`json:"virtual_host"`
		Priority	int			`json:"priority"`
		PathPrefix	string			`json:"path_prefix"`
		PathExact	string			`json:"path_exact"`
		Methods		[]string		`json:"methods"`
		Headers		map[string]string	`json:"headers"`
		Upstream	string			`json:"upstream"`
		Timeout		string			`json:"timeout"`
		RetryCount	int			`json:"retry_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.VirtualHost == "" || body.Upstream == "" {
		http.Error(w, "virtual_host and upstream are required", http.StatusBadRequest)
		return
	}

	timeout, err := parseTimeout(body.Timeout)
	if err != nil {
		http.Error(w, "invalid timeout: "+err.Error(), http.StatusBadRequest)
		return
	}

	route := &types.Route{
		ID:		id,
		Priority:	body.Priority,
		Match: types.RouteMatch{
			PathPrefix:	body.PathPrefix,
			PathExact:	body.PathExact,
			Methods:	body.Methods,
			Headers:	body.Headers,
		},
		Action: types.RouteAction{
			UpstreamPool:	body.Upstream,
			Timeout:	timeout,
			RetryCount:	body.RetryCount,
		},
	}

	cfg := cloneConfig(a.store.Get())
	addOrUpdateRouteInConfig(cfg, route, body.VirtualHost)
	a.store.Set(cfg)
	applyRouteToRouter(a.router, cfg, route, body.VirtualHost)

	writeJSON(w, http.StatusOK, map[string]any{"message": "route updated", "id": id})
}

func (a *AdminAPI) deleteRoute(w http.ResponseWriter, r *http.Request, id string) {
	cfg := cloneConfig(a.store.Get())
	removed := deleteRouteFromConfig(cfg, id)
	if !removed {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}
	a.store.Set(cfg)
	rebuildRouterForConfig(a.router, cfg)
	writeJSON(w, http.StatusOK, map[string]any{"message": "route removed", "id": id})
}

func (a *AdminAPI) listUpstreams(w http.ResponseWriter, r *http.Request) {
	type endpointResponse struct {
		ID	string	`json:"id"`
		Address	string	`json:"address"`
		Weight	int	`json:"weight"`
		Healthy	bool	`json:"healthy"`
	}
	type upstreamResponse struct {
		Name		string			`json:"name"`
		Endpoints	[]endpointResponse	`json:"endpoints"`
	}

	var upstreams []upstreamResponse
	cfg := a.store.Get()

	for _, up := range cfg.Upstreams {
		lb, ok := a.balancers[up.Name]
		if !ok {
			continue
		}

		var endpoints []endpointResponse
		if inspector, ok := lb.(interface{ Endpoints() []*types.Endpoint }); ok {
			for _, ep := range inspector.Endpoints() {
				endpoints = append(endpoints, endpointResponse{
					ID:		ep.ID,
					Address:	ep.Address,
					Weight:		ep.Weight,
					Healthy:	ep.Healthy,
				})
			}
		}

		upstreams = append(upstreams, upstreamResponse{
			Name:		up.Name,
			Endpoints:	endpoints,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"upstreams": upstreams, "count": len(upstreams)})
}

func (a *AdminAPI) addEndpoint(w http.ResponseWriter, r *http.Request, poolName string) {
	var body struct {
		ID	string	`json:"id"`
		Address	string	`json:"address"`
		Weight	int	`json:"weight"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.ID == "" || body.Address == "" {
		http.Error(w, "id and address are required", http.StatusBadRequest)
		return
	}

	lb, ok := a.balancers[poolName]
	if !ok {
		http.Error(w, "upstream pool not found: "+poolName, http.StatusNotFound)
		return
	}

	endpoint := &types.Endpoint{
		ID:		body.ID,
		Address:	body.Address,
		Weight:		defaultWeight(body.Weight),
		Healthy:	true,
	}

	cfg := cloneConfig(a.store.Get())
	if err := updateUpstreamEndpointsInConfig(cfg, poolName, endpoint, true); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lb.Add(endpoint)
	a.store.Set(cfg)
	writeJSON(w, http.StatusCreated, map[string]any{"message": "endpoint added", "id": body.ID, "pool": poolName})
}

func (a *AdminAPI) removeEndpoint(w http.ResponseWriter, r *http.Request, poolName, endpointID string) {
	lb, ok := a.balancers[poolName]
	if !ok {
		http.Error(w, "upstream pool not found: "+poolName, http.StatusNotFound)
		return
	}

	cfg := cloneConfig(a.store.Get())
	if err := updateUpstreamEndpointsInConfig(cfg, poolName, &types.Endpoint{ID: endpointID}, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lb.Remove(endpointID)
	a.store.Set(cfg)
	writeJSON(w, http.StatusOK, map[string]any{"message": "endpoint removed", "id": endpointID, "pool": poolName})
}

func (a *AdminAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := a.store.Get()
	routeCount := 0
	for _, vh := range cfg.VirtualHosts {
		routeCount += len(vh.Routes)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":		"ok",
		"virtual_hosts":	len(cfg.VirtualHosts),
		"routes":		routeCount,
		"upstreams":		len(cfg.Upstreams),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseTimeout(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	return time.ParseDuration(value)
}

func defaultWeight(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func cloneConfig(cfg *config.OusiaConfig) *config.OusiaConfig {
	clone := &config.OusiaConfig{
		Gateway: cfg.Gateway,
	}
	for _, vh := range cfg.VirtualHosts {
		newVh := config.VirtualHostConfig{
			Hostname:	vh.Hostname,
			TLS:		vh.TLS,
		}
		for _, route := range vh.Routes {
			newVh.Routes = append(newVh.Routes, config.RouteConfig{
				ID:       route.ID,
				Priority: route.Priority,
				Match: config.MatchConfig{
					PathPrefix: route.Match.PathPrefix,
					PathExact:  route.Match.PathExact,
					Methods:    append([]string(nil), route.Match.Methods...),
					Headers:    copyStringMap(route.Match.Headers),
				},
				Action: config.ActionConfig{
					Upstream:          route.Action.Upstream,
					StripPrefix:       route.Action.StripPrefix,
					AddHeaders:        copyStringMap(route.Action.AddHeaders),
					RemoveHeaders:     append([]string(nil), route.Action.RemoveHeaders...),
					AddRespHeaders:    copyStringMap(route.Action.AddRespHeaders),
					RemoveRespHeaders: append([]string(nil), route.Action.RemoveRespHeaders...),
					Timeout:           route.Action.Timeout,
					RetryCount:        route.Action.RetryCount,
				},
			})
		}
		clone.VirtualHosts = append(clone.VirtualHosts, newVh)
	}
	for _, up := range cfg.Upstreams {
		newUp := config.UpstreamConfig{
			Name:		up.Name,
			Algorithm:	up.Algorithm,
		}
		for _, ep := range up.Endpoints {
			newUp.Endpoints = append(newUp.Endpoints, config.EndpointConfig{
				ID:		ep.ID,
				Address:	ep.Address,
				Weight:		ep.Weight,
				Meta:		copyStringMap(ep.Meta),
			})
		}
		clone.Upstreams = append(clone.Upstreams, newUp)
	}
	return clone
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func findRouteIndexes(cfg *config.OusiaConfig, id string) (int, int, *config.RouteConfig) {
	for vi, vh := range cfg.VirtualHosts {
		for ri, route := range vh.Routes {
			if route.ID == id {
				return vi, ri, &cfg.VirtualHosts[vi].Routes[ri]
			}
		}
	}
	return -1, -1, nil
}

func deleteRouteFromConfig(cfg *config.OusiaConfig, id string) bool {
	for vi, vh := range cfg.VirtualHosts {
		for ri, route := range vh.Routes {
			if route.ID != id {
				continue
			}
			vh.Routes = append(vh.Routes[:ri], vh.Routes[ri+1:]...)
			if len(vh.Routes) == 0 {
				cfg.VirtualHosts = append(cfg.VirtualHosts[:vi], cfg.VirtualHosts[vi+1:]...)
			} else {
				cfg.VirtualHosts[vi] = vh
			}
			return true
		}
	}
	return false
}

func addOrUpdateRouteInConfig(cfg *config.OusiaConfig, route *types.Route, virtualHost string) {
	deleteRouteFromConfig(cfg, route.ID)

	for vi, vh := range cfg.VirtualHosts {
		if vh.Hostname != virtualHost {
			continue
		}
		vh.Routes = append(vh.Routes, config.RouteConfig{
			ID:       route.ID,
			Priority: route.Priority,
			Match: config.MatchConfig{
				PathPrefix: route.Match.PathPrefix,
				PathExact:  route.Match.PathExact,
				Methods:    append([]string(nil), route.Match.Methods...),
				Headers:    copyStringMap(route.Match.Headers),
			},
			Action: config.ActionConfig{
				Upstream:          route.Action.UpstreamPool,
				StripPrefix:       route.Action.StripPrefix,
				AddHeaders:        copyStringMap(route.Action.AddHeaders),
				RemoveHeaders:     append([]string(nil), route.Action.RemoveHeaders...),
				AddRespHeaders:    copyStringMap(route.Action.AddRespHeaders),
				RemoveRespHeaders: append([]string(nil), route.Action.RemoveRespHeaders...),
				Timeout:           route.Action.Timeout.String(),
				RetryCount:        route.Action.RetryCount,
			},
		})
		sort.Slice(vh.Routes, func(i, j int) bool {
			return vh.Routes[i].Priority < vh.Routes[j].Priority
		})
		cfg.VirtualHosts[vi] = vh
		return
	}

	newVh := config.VirtualHostConfig{
		Hostname: virtualHost,
		Routes: []config.RouteConfig{{
			ID:       route.ID,
			Priority: route.Priority,
			Match: config.MatchConfig{
				PathPrefix: route.Match.PathPrefix,
				PathExact:  route.Match.PathExact,
				Methods:    append([]string(nil), route.Match.Methods...),
				Headers:    copyStringMap(route.Match.Headers),
			},
			Action: config.ActionConfig{
				Upstream:          route.Action.UpstreamPool,
				StripPrefix:       route.Action.StripPrefix,
				AddHeaders:        copyStringMap(route.Action.AddHeaders),
				RemoveHeaders:     append([]string(nil), route.Action.RemoveHeaders...),
				AddRespHeaders:    copyStringMap(route.Action.AddRespHeaders),
				RemoveRespHeaders: append([]string(nil), route.Action.RemoveRespHeaders...),
				Timeout:           route.Action.Timeout.String(),
				RetryCount:        route.Action.RetryCount,
			},
		}},
	}
	cfg.VirtualHosts = append(cfg.VirtualHosts, newVh)
}

func updateUpstreamEndpointsInConfig(cfg *config.OusiaConfig, poolName string, endpoint *types.Endpoint, add bool) error {
	for ui, up := range cfg.Upstreams {
		if up.Name != poolName {
			continue
		}
		if add {
			for _, ep := range up.Endpoints {
				if ep.ID == endpoint.ID {
					return fmt.Errorf("endpoint %q already exists in pool %q", endpoint.ID, poolName)
				}
			}
			up.Endpoints = append(up.Endpoints, config.EndpointConfig{
				ID:		endpoint.ID,
				Address:	endpoint.Address,
				Weight:		endpoint.Weight,
				Meta:		copyStringMap(endpoint.Metadata),
			})
			cfg.Upstreams[ui] = up
			return nil
		}
		for ei, ep := range up.Endpoints {
			if ep.ID == endpoint.ID {
				up.Endpoints = append(up.Endpoints[:ei], up.Endpoints[ei+1:]...)
				cfg.Upstreams[ui] = up
				return nil
			}
		}
		return fmt.Errorf("endpoint %q not found in pool %q", endpoint.ID, poolName)
	}
	return fmt.Errorf("upstream pool not found: %s", poolName)
}

func applyRouteToRouter(r *router.Router, cfg *config.OusiaConfig, route *types.Route, virtualHost string) {
	for _, vhCfg := range cfg.VirtualHosts {
		if vhCfg.Hostname != virtualHost {
			continue
		}
		vh := &types.VirtualHost{
			Hostname: vhCfg.Hostname,
		}
		for _, rCfg := range vhCfg.Routes {
			vh.Routes = append(vh.Routes, &types.Route{
				ID:		rCfg.ID,
				Priority:	rCfg.Priority,
				Match: types.RouteMatch{
					PathPrefix:	rCfg.Match.PathPrefix,
					PathExact:	rCfg.Match.PathExact,
					Methods:	append([]string(nil), rCfg.Match.Methods...),
					Headers:	copyStringMap(rCfg.Match.Headers),
				},
				Action: types.RouteAction{
					UpstreamPool:	rCfg.Action.Upstream,
					StripPrefix:	rCfg.Action.StripPrefix,
					AddHeaders:	copyStringMap(rCfg.Action.AddHeaders),
					Timeout:	parseDurationOrZero(rCfg.Action.Timeout),
					RetryCount:	rCfg.Action.RetryCount,
				},
			})
		}
		r.AddVirtualHost(vh)
		return
	}
}

func rebuildRouterForConfig(r *router.Router, cfg *config.OusiaConfig) {
	for _, vhCfg := range cfg.VirtualHosts {
		vh := &types.VirtualHost{
			Hostname: vhCfg.Hostname,
		}
		for _, rCfg := range vhCfg.Routes {
			vh.Routes = append(vh.Routes, &types.Route{
				ID:		rCfg.ID,
				Priority:	rCfg.Priority,
				Match: types.RouteMatch{
					PathPrefix:	rCfg.Match.PathPrefix,
					PathExact:	rCfg.Match.PathExact,
					Methods:	append([]string(nil), rCfg.Match.Methods...),
					Headers:	copyStringMap(rCfg.Match.Headers),
				},
				Action: types.RouteAction{
					UpstreamPool:	rCfg.Action.Upstream,
					StripPrefix:	rCfg.Action.StripPrefix,
					AddHeaders:	copyStringMap(rCfg.Action.AddHeaders),
					Timeout:	parseDurationOrZero(rCfg.Action.Timeout),
					RetryCount:	rCfg.Action.RetryCount,
				},
			})
		}
		r.AddVirtualHost(vh)
	}
}

func parseDurationOrZero(value string) time.Duration {
	if value == "" {
		return 0
	}
	d, _ := time.ParseDuration(value)
	return d
}

func (a *AdminAPI) handleMeshRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ServiceID	string			`json:"service_id"`
		InstanceID	string			`json:"instance_id"`
		Address		string			`json:"address"`
		Port		int			`json:"port"`
		Meta		map[string]string	`json:"meta,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.ServiceID == "" || body.InstanceID == "" || body.Address == "" || body.Port == 0 {
		http.Error(w, "service_id, instance_id, address, and port are required", http.StatusBadRequest)
		return
	}

	inst := &ServiceInstance{
		ServiceID:	body.ServiceID,
		InstanceID:	body.InstanceID,
		Address:	body.Address,
		Port:		body.Port,
		Meta:		body.Meta,
	}

	a.mesh.Register(inst)
	fmt.Printf("mesh: registered instance %q for service %q at %s:%d\n", inst.InstanceID, inst.ServiceID, inst.Address, inst.Port)
	writeJSON(w, http.StatusCreated, map[string]any{"message": "registered", "instance_id": inst.InstanceID})
}

func (a *AdminAPI) handleMeshHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	if !a.mesh.Heartbeat(body.InstanceID) {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "heartbeat acknowledged", "instance_id": body.InstanceID})
}

func (a *AdminAPI) handleMeshDeregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	if !a.mesh.Deregister(body.InstanceID) {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	fmt.Printf("mesh: deregistered instance %q\n", body.InstanceID)
	writeJSON(w, http.StatusOK, map[string]any{"message": "deregistered", "instance_id": body.InstanceID})
}

func (a *AdminAPI) handleMeshServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type instanceResponse struct {
		InstanceID	string			`json:"instance_id"`
		Address		string			`json:"address"`
		Port		int			`json:"port"`
		Meta		map[string]string	`json:"meta,omitempty"`
		LastHeartbeat	string			`json:"last_heartbeat"`
	}
	type serviceResponse struct {
		ServiceID	string			`json:"service_id"`
		Instances	[]instanceResponse	`json:"instances"`
	}

	serviceMap := make(map[string][]instanceResponse)
	for _, inst := range a.mesh.Instances() {
		serviceMap[inst.ServiceID] = append(serviceMap[inst.ServiceID], instanceResponse{
			InstanceID:	inst.InstanceID,
			Address:	inst.Address,
			Port:		inst.Port,
			Meta:		inst.Meta,
			LastHeartbeat:	inst.LastHeartbeat.Format(time.RFC3339),
		})
	}

	var services []serviceResponse
	for svcID, instances := range serviceMap {
		services = append(services, serviceResponse{
			ServiceID:	svcID,
			Instances:	instances,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"services": services, "count": len(services)})
}

func (a *AdminAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gatewayStats := observability.GetStatsJSON()
	sidecarStats := make(map[string]map[string]string)

	client := &http.Client{Timeout: 2 * time.Second}

	for _, inst := range a.mesh.Instances() {
		if _, ok := sidecarStats[inst.ServiceID]; !ok {
			sidecarStats[inst.ServiceID] = make(map[string]string)
		}

		url := fmt.Sprintf("http://%s:%d/stats", inst.Address, inst.Port)
		resp, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			sidecarStats[inst.ServiceID][inst.InstanceID] = string(body)
		} else {
			sidecarStats[inst.ServiceID][inst.InstanceID] = "error: " + err.Error()
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"gateway":	gatewayStats,
		"sidecars":	sidecarStats,
	})
}
