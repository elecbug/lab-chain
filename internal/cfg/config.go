package cfg

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string        `yaml:"log_level"`
	Mode     string        `yaml:"mode"` // e.g., "full", "light", "boot"
	Network  NetworkConfig `yaml:"network"`
	DHT      DHTConfig     `yaml:"dht"`
}

type NetworkConfig struct {
	IPAddress string `yaml:"ip_address"`
	MaxPeers  int    `yaml:"max_peers"` // Maximum number of peers to connect to
}

type DHTConfig struct {
	Mode           string   `yaml:"mode"`            // e.g., "server", "client"
	BootstrapPeers []string `yaml:"bootstrap_peers"` // List of bootstrap peers for DHT
}

// initCfg initializes the configuration from the YAML file
func InitCfg() Config {
	cfg := flag.String("cfg", "cfg.yaml", "Path to the configuration file")
	flag.Parse()

	file, err := os.Open(*cfg)

	if err != nil {
		log.Fatalf("Failed to open configuration file: %v", err)
	}
	defer file.Close()

	var config Config

	decoder := yaml.NewDecoder(file)

	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to decode YAML file into cfg.Config: %v", err)
	}

	log.Printf("Configuration: %+v", config)

	return config
}
