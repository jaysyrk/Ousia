package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaysyrk/ousia/internal/gateway"
	"github.com/jaysyrk/ousia/pkg/types"
)

func main() {
	cfg := &types.SidecarConfig{
		ServiceID:    "service-a",
		InboundPort:  15000,
		OutboundPort: 15001,
		LocalPort:    8080,
		Upstreams: []types.SidecarUpstream{
			{Name: "service-b", Address: "10.0.0.2:15000"},
		},
	}

	inbound := gateway.NewInboundProxy(cfg.LocalPort)
	upstreamMap := make(map[string]string)
	for _, u := range cfg.Upstreams {
		upstreamMap[u.Name] = u.Address
	}
	outbound := gateway.NewOutboundProxy(upstreamMap)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := inbound.Start(fmt.Sprintf(":%d", cfg.InboundPort)); err != nil {
			log.Printf("inbound proxy error: %v", err)
		}
	}()

	go func() {
		if err := outbound.Start(fmt.Sprintf(":%d", cfg.OutboundPort)); err != nil {
			log.Printf("outbound proxy error: %v", err)
		}
	}()

	fmt.Printf("Ousia sidecar started for service %q\n", cfg.ServiceID)
	<-quit
	fmt.Println("sidecar shutting down")
	os.Exit(0)
}
