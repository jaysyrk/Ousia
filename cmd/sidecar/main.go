package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaysyrk/ousia/internal/gateway"
	"github.com/jaysyrk/ousia/pkg/types"
)

func main() {
	serviceID := flag.String("service-id", "service-a", "service id")
	localPort := flag.Int("local-port", 8080, "local service port")
	inboundPort := flag.Int("inbound-port", 15000, "sidecar inbound port")
	outboundPort := flag.Int("outbound-port", 15001, "sidecar outbound port")
	adminURL := flag.String("admin-url", "http://127.0.0.1:9000", "gateway admin API URL")
	refreshInterval := flag.Duration("refresh-interval", 5*time.Second, "upstream refresh interval")
	flag.Parse()

	if *adminURL == "" {
		fmt.Println("admin-url is required")
		os.Exit(1)
	}

	cfg := &types.SidecarConfig{
		ServiceID:       *serviceID,
		InboundPort:     *inboundPort,
		OutboundPort:    *outboundPort,
		LocalPort:       *localPort,
		AdminURL:        *adminURL,
		RefreshInterval: *refreshInterval,
	}

	// Generate a unique instance ID for mesh registration
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%s-%d", cfg.ServiceID, hostname, cfg.InboundPort)

	mapper := gateway.NewServiceMapper()
	discovery := gateway.NewSidecarDiscovery(cfg.AdminURL, mapper, cfg.RefreshInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go discovery.Start(ctx)

	// Register with the service mesh
	registrar := gateway.NewSidecarRegistrar(
		cfg.AdminURL,
		cfg.ServiceID,
		instanceID,
		"127.0.0.1",
		cfg.InboundPort,
		10*time.Second,
	)
	go registrar.Start(ctx)

	inbound := gateway.NewInboundProxy(cfg.LocalPort)
	outbound := gateway.NewOutboundProxy(mapper)

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

	fmt.Printf("Ousia sidecar started for service %q (instance: %s)\n", cfg.ServiceID, instanceID)
	<-quit
	fmt.Println("sidecar shutting down, deregistering from mesh...")
	cancel() // triggers deregister in registrar
	time.Sleep(500 * time.Millisecond) // brief grace period for deregister call
}
