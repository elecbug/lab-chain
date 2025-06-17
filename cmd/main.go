package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/cli"
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
		Chain:          nil,
		TxTopic:        txTopic,
		BlockTopic:     blkTopic,
		MemPool:        chain.NewMempool(),
		CurrentPrivKey: nil,
		CurrentAddress: nil,
	}

	cliCommand(&user)

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

// cliCommand defines the command-line interface for blockchain operations
func cliCommand(user *user.User) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Cli started. Type 'help' to see available commands.\n")

	for {
		fmt.Print("$ ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "help":
		case "exit":
			return
		case "mkey":
			cli.MkeyFunc(user, args)
		case "wallet":
			cli.WalletFunc(user, args)
		case "tx":
			cli.TxFunc(user, args)
		case "mine":
			cli.MineFunc(user, args)
		case "chain":
			cli.ChainFunc(user, args)

		default:
			fmt.Printf("Unknown command. Type 'help' for options.\n")
		}
	}
}
