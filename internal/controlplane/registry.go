package controlplane

import (
	"sync"
	"time"
)

// ServiceInstance represents a single sidecar-registered service instance.
type ServiceInstance struct {
	ServiceID   string            `json:"service_id"`
	InstanceID  string            `json:"instance_id"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Meta        map[string]string `json:"meta,omitempty"`
	RegisteredAt time.Time        `json:"registered_at"`
	LastHeartbeat time.Time       `json:"last_heartbeat"`
}

// MeshRegistry tracks all active sidecar-registered service instances.
type MeshRegistry struct {
	mu        sync.RWMutex
	instances map[string]*ServiceInstance // key = instanceID
	ttl       time.Duration
}

// NewMeshRegistry creates a new registry with the given heartbeat TTL.
// Instances that haven't sent a heartbeat within the TTL are considered stale.
func NewMeshRegistry(ttl time.Duration) *MeshRegistry {
	return &MeshRegistry{
		instances: make(map[string]*ServiceInstance),
		ttl:       ttl,
	}
}

// Register adds or updates a service instance in the registry.
func (r *MeshRegistry) Register(inst *ServiceInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if existing, ok := r.instances[inst.InstanceID]; ok {
		// Update existing registration
		existing.Address = inst.Address
		existing.Port = inst.Port
		existing.Meta = inst.Meta
		existing.LastHeartbeat = now
		return
	}

	inst.RegisteredAt = now
	inst.LastHeartbeat = now
	r.instances[inst.InstanceID] = inst
}

// Heartbeat refreshes the TTL for an instance. Returns false if instance not found.
func (r *MeshRegistry) Heartbeat(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.instances[instanceID]
	if !ok {
		return false
	}
	inst.LastHeartbeat = time.Now()
	return true
}

// Deregister removes an instance from the registry.
func (r *MeshRegistry) Deregister(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.instances[instanceID]
	if ok {
		delete(r.instances, instanceID)
	}
	return ok
}

// Instances returns all live instances, evicting any that have exceeded the TTL.
func (r *MeshRegistry) Instances() []*ServiceInstance {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var live []*ServiceInstance

	for id, inst := range r.instances {
		if now.Sub(inst.LastHeartbeat) > r.ttl {
			delete(r.instances, id)
			continue
		}
		live = append(live, inst)
	}

	return live
}

// InstancesByService returns all live instances matching the given service ID.
func (r *MeshRegistry) InstancesByService(serviceID string) []*ServiceInstance {
	all := r.Instances()
	var out []*ServiceInstance
	for _, inst := range all {
		if inst.ServiceID == serviceID {
			out = append(out, inst)
		}
	}
	return out
}

// ServiceNames returns a deduplicated list of all registered service IDs.
func (r *MeshRegistry) ServiceNames() []string {
	all := r.Instances()
	seen := make(map[string]bool)
	var names []string
	for _, inst := range all {
		if !seen[inst.ServiceID] {
			seen[inst.ServiceID] = true
			names = append(names, inst.ServiceID)
		}
	}
	return names
}
