package logging

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/cfg"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

// InitLogging initializes the logging system for the application.
func InitLogging(h host.Host, cfg cfg.Config) error {
	level, err := ipfslog.LevelFromString(cfg.LogLevel)

	if err != nil {
		return fmt.Errorf("failed to set log level: %v", err)
	}

	labels := map[string]string{
		"peerID": h.ID().String(),
	}

	ipfslog.SetupLogging(ipfslog.Config{
		Level:  level,
		Labels: labels,
		Format: ipfslog.JSONOutput,
		Stdout: false,
		Stderr: false,
		File:   "/app/data/log.jsonl",
	})

	return nil
}
