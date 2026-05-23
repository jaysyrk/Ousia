package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SidecarRegistrar struct {
	adminURL	string
	serviceID	string
	instanceID	string
	address		string
	port		int
	interval	time.Duration
	client		*http.Client
}

func NewSidecarRegistrar(adminURL, serviceID, instanceID, address string, port int, interval time.Duration) *SidecarRegistrar {
	return &SidecarRegistrar{
		adminURL:	adminURL,
		serviceID:	serviceID,
		instanceID:	instanceID,
		address:	address,
		port:		port,
		interval:	interval,
		client:		&http.Client{Timeout: 5 * time.Second},
	}
}

func (r *SidecarRegistrar) Start(ctx context.Context) {
	r.register()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.deregister()
			return
		case <-ticker.C:
			r.heartbeat()
		}
	}
}

func (r *SidecarRegistrar) register() {
	body := map[string]any{
		"service_id":	r.serviceID,
		"instance_id":	r.instanceID,
		"address":	r.address,
		"port":		r.port,
	}

	data, _ := json.Marshal(body)
	resp, err := r.client.Post(r.adminURL+"/api/mesh/register", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("sidecar registrar: registration failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("sidecar registrar: registered as %q for service %q\n", r.instanceID, r.serviceID)
	} else {
		fmt.Printf("sidecar registrar: unexpected status %d on register\n", resp.StatusCode)
	}
}

func (r *SidecarRegistrar) heartbeat() {
	body := map[string]any{
		"instance_id": r.instanceID,
	}

	data, _ := json.Marshal(body)
	resp, err := r.client.Post(r.adminURL+"/api/mesh/heartbeat", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("sidecar registrar: heartbeat failed: %v\n", err)

		r.register()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {

		fmt.Println("sidecar registrar: instance evicted, re-registering")
		r.register()
	}
}

func (r *SidecarRegistrar) deregister() {
	body := map[string]any{
		"instance_id": r.instanceID,
	}

	data, _ := json.Marshal(body)
	resp, err := r.client.Post(r.adminURL+"/api/mesh/deregister", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("sidecar registrar: deregister failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("sidecar registrar: deregistered %q\n", r.instanceID)
}
