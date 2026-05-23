package main

import (
	"fmt"
	"log"

	"github.com/jaysyrk/ousia/pkg/config"
)

func main() {
	cfg, err := config.Load("ousia.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("Ousia Gateway starting on %s\n", cfg.Gateway.ListenAddr)
	fmt.Printf("Loaded %d virtual host(s)\n", len(cfg.VirtualHosts))
	fmt.Printf("Loaded %d upstream pool(s)\n", len(cfg.Upstreams))
}