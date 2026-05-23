package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaysyrk/ousia/internal/gateway"
	"github.com/jaysyrk/ousia/pkg/config"
)

func main() {
	configPath := "ousia.yaml"

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, err := gateway.Bootstrap(cfg, configPath)
	if err != nil {
		log.Fatalf("failed to bootstrap gateway: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("gateway stopped: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
}
