package main

import (
	"context"
	"log"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/libp2p"
)

func main() {
	// Initialize configuration from the YAML file
	cfg := cfg.InitCfg()
	ctx := context.Background()

	// Set up the libp2p host using the configuration
	h, err := libp2p.SetLibp2pHost(cfg)

	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}

	// Set up the Kademlia DHT for peer discovery and routing
	_, err = libp2p.SetKadDHT(ctx, h, cfg)

	if err != nil {
		log.Fatalf("Failed to create Kademlia DHT: %v", err)
	}

	_, _, err = libp2p.SetGossipSub(ctx, h)

	if err != nil {
		log.Fatalf("Failed to create GossipSub: %v", err)
	}

	log.Println("Libp2p host, DHT, and GossipSub initialized successfully")

	for _, addr := range h.Addrs() {
		log.Printf("Libp2p host listening on: %s", addr)
	}

	select {}
}
