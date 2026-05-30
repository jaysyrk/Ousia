package controlplane

import (
	"context"
	"testing"
	"time"
)

func TestMeshRegistry(t *testing.T) {
	reg := NewMeshRegistry(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reg.StartCleanup(ctx)

	inst := &ServiceInstance{
		ServiceID:  "svc-1",
		InstanceID: "inst-1",
		Address:    "127.0.0.1",
		Port:       8080,
	}

	reg.Register(inst)
	if len(reg.Instances()) != 1 {
		t.Error("expected 1 instance")
	}

	if len(reg.InstancesByService("svc-1")) != 1 {
		t.Error("expected 1 instance for svc-1")
	}

	if len(reg.ServiceNames()) != 1 {
		t.Error("expected 1 service name")
	}

	// Update existing
	inst.Port = 8081
	reg.Register(inst)

	// Heartbeat
	if !reg.Heartbeat("inst-1") {
		t.Error("expected heartbeat to succeed")
	}
	if reg.Heartbeat("non-existent") {
		t.Error("expected heartbeat to fail")
	}

	// Deregister
	if !reg.Deregister("inst-1") {
		t.Error("expected deregister to succeed")
	}
	if len(reg.Instances()) != 0 {
		t.Error("expected 0 instances")
	}

	// Cleanup test
	reg.Register(inst)
	time.Sleep(100 * time.Millisecond) // wait for ttl
	if len(reg.Instances()) != 0 {
		t.Error("expected 0 instances after ttl")
	}
}
