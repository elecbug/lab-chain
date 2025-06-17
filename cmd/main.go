package main

import (
	"context"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/node"
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

		if err := node.InitGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	case "light":
		log.Infof("running in light node mode")

		if err := node.InitGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	case "boot":
		log.Infof("running in boot node mode")

		if err := node.InitBootNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	default:
		log.Infof("unknown mode %s, defaulting to light node mode", cfg.Mode)
		cfg.Mode = "light"

		if err := node.InitGeneralNode(ctx, *cfg, *priv); err != nil {
			log.Fatalw("failed to initialize full node: %v", err)
		}
	}
}
