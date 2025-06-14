package main

import (
	"context"
	"log"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/libp2p"
	"github.com/elecbug/lab-chain/internal/logging"
)

func main() {
	// Initialize configuration from the YAML file
	cfg := cfg.InitCfg()
	ctx := context.Background()

	switch cfg.Mode {
	case "full":
		log.Println("Running in full node mode")
		initGeneralNode(ctx, cfg)
	case "light":
		log.Println("Running in light node mode")
		initGeneralNode(ctx, cfg)
	case "boot":
		log.Println("Running in boot node mode")
		initBootNode(ctx, cfg)
	default:
		log.Printf("Unknown mode %s, defaulting to light node mode", cfg.Mode)
		cfg.Mode = "light"
		initGeneralNode(ctx, cfg)
	}
}

func initGeneralNode(ctx context.Context, cfg cfg.Config) {
	// This function can be used to initialize a general node setup
	// It can include additional configurations or services as needed
	log.Println("Initializing general node setup...")
	// Additional initialization logic can go here

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

	log.Println("Libp2p host listening on:")
	for _, addr := range h.Addrs() {
		log.Printf("%s/p2p/%s", addr, h.ID())
	}

	logging.InitLogging(h, cfg)

	log.Println("Logging initialized with level:", cfg.LogLevel)

	select {}
}

func initBootNode(ctx context.Context, cfg cfg.Config) {
	// This function can be used to initialize a boot node setup
	// It can include additional configurations or services as needed
	log.Println("Initializing boot node setup...")
	// Additional initialization logic can go here

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

	log.Println("Libp2p host, DHT, and GossipSub initialized successfully")

	log.Printf("Libp2p host listening on:")
	for _, addr := range h.Addrs() {
		log.Printf("- %s/p2p/%s", addr, h.ID())
	}

	logging.InitLogging(h, cfg)

	log.Println("Logging initialized with level:", cfg.LogLevel)

	select {}
}
