package logging

import (
	"log"

	"github.com/elecbug/lab-chain/internal/cfg"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

// InitLogging initializes the logging system for the application.
func InitLogging(h host.Host, cfg cfg.Config) {
	level, err := ipfslog.LevelFromString(cfg.LogLevel)

	if err != nil {
		log.Fatalf("Failed to set log level: %v", err)
	}

	labels := map[string]string{
		"peerID": h.ID().String(),
	}

	ipfslog.SetupLogging(ipfslog.Config{
		Level:  level,
		Labels: labels,
		Format: ipfslog.JSONOutput,
		Stdout: true,
		Stderr: true,
		File:   "/app/data/log.jsonl",
	})
}
