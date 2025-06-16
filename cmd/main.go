package main

import (
	"context"
	"fmt"

	"github.com/elecbug/lab-chain/internal/blockchain"
	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/libp2p"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/logging"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	log := logger.AppLogger

	// Initialize configuration from the YAML file
	cfg, priv, err := cfg.InitSetting()

	if err != nil {
		log.Fatalw("failed to initialize setting: %v", err)
	} else {
		log.Infof("setting initialized successfully")
	}

	ctx := context.Background()

	switch cfg.Mode {
	case "full":
		log.Infof("running in full node mode")

		if err := initGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	case "light":
		log.Infof("running in light node mode")

		if err := initGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	case "boot":
		log.Infof("running in boot node mode")

		if err := initBootNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	default:
		log.Infof("unknown mode %s, defaulting to light node mode", cfg.Mode)
		cfg.Mode = "light"

		if err := initGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	}
}

func initGeneralNode(ctx context.Context, cfg cfg.Config, priv crypto.PrivKey) error {
	log := logger.AppLogger
	// This function can be used to initialize a general node setup
	// It can include additional configurations or services as needed
	// Additional initialization logic can go here

	// Set up the libp2p host using the configuration
	h, err := libp2p.SetLibp2pHost(cfg, priv)

	logging.InitLogging(h, cfg)

	log.Infof("logging initialized with level: %s", cfg.LogLevel)
	log.Infof("initializing general node setup")

	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Set up the Kademlia DHT for peer discovery and routing
	_, err = libp2p.SetKadDHT(ctx, h, cfg)

	if err != nil {
		return fmt.Errorf("failed to create Kademlia DHT: %v", err)
	}

	blkTopic, txTopic, err := libp2p.SetGossipSub(ctx, h)

	if err != nil {
		return fmt.Errorf("failed to create GossipSub: %v", err)
	}

	log.Infof("libp2p host, DHT, and GossipSub initialized successfully")

	addrs := make([]string, 0)
	for _, addr := range h.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, h.ID()))
	}

	log.Infof("libp2p host listening on %v", addrs)

	user := user.User{
		Context:        ctx,
		MasterKey:      nil,
		Blockchain:     nil,
		TxTopic:        txTopic,
		BlockTopic:     blkTopic,
		MemPool:        blockchain.NewMempool(),
		CurrentPrivKey: nil,
		CurrentAddress: nil,
	}

	CLICommand(&user)

	return nil
}

func initBootNode(ctx context.Context, cfg cfg.Config, priv crypto.PrivKey) error {
	log := logger.AppLogger

	// This function can be used to initialize a boot node setup
	// It can include additional configurations or services as needed
	// Additional initialization logic can go here

	// Set up the libp2p host using the configuration
	h, err := libp2p.SetLibp2pHost(cfg, priv)

	logging.InitLogging(h, cfg)

	log.Infof("logging initialized with level: %s", cfg.LogLevel)
	log.Infof("initializing general node setup")

	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Set up the Kademlia DHT for peer discovery and routing
	_, err = libp2p.SetKadDHT(ctx, h, cfg)

	if err != nil {
		return fmt.Errorf("failed to create Kademlia DHT: %v", err)
	}

	log.Infof("libp2p host, DHT, and GossipSub initialized successfully")

	addrs := make([]string, 0)
	for _, addr := range h.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, h.ID()))
	}

	log.Infof("libp2p host listening on %v", addrs)

	select {}
}
