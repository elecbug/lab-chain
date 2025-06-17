package node

import (
	"context"
	"fmt"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/cli"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/logging"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func InitGeneralNode(ctx context.Context, cfg cfg.Config, priv crypto.PrivKey) error {
	log := logger.AppLogger

	h, err := setLibp2pHost(cfg, priv)

	logging.InitLogging(h, cfg)

	log.Infof("logging initialized with level: %s", cfg.LogLevel)
	log.Infof("initializing general node setup")

	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Set up the Kademlia DHT for peer discovery and routing
	_, err = setKadDHT(ctx, h, cfg)

	if err != nil {
		return fmt.Errorf("failed to create Kademlia DHT: %v", err)
	}

	blkTopic, txTopic, err := setGossipSub(ctx, h)

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
		Chain:          nil,
		TxTopic:        txTopic,
		BlockTopic:     blkTopic,
		MemPool:        chain.NewMempool(),
		CurrentPrivKey: nil,
		CurrentAddress: nil,
	}

	cli.CliCommand(&user)

	return nil
}

func InitBootNode(ctx context.Context, cfg cfg.Config, priv crypto.PrivKey) error {
	log := logger.AppLogger

	h, err := setLibp2pHost(cfg, priv)

	logging.InitLogging(h, cfg)

	log.Infof("logging initialized with level: %s", cfg.LogLevel)
	log.Infof("initializing general node setup")

	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Set up the Kademlia DHT for peer discovery and routing
	_, err = setKadDHT(ctx, h, cfg)

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
