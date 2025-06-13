package cfg

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string        `yaml:"log_level"`
	Network  NetworkConfig `yaml:"network"`
	DHT      DHTConfig     `yaml:"dht"`
}

type NetworkConfig struct {
	IPAddress string `yaml:"ip_address"`
	Port      int    `yaml:"port"`
	MaxPeers  int    `yaml:"max_peers"` // Maximum number of peers to connect to
}

type DHTConfig struct {
	Mode           string   `yaml:"mode"`            // e.g., "server", "client"
	BootstrapPeers []string `yaml:"bootstrap_peers"` // List of bootstrap peers for DHT
}

// initCfg initializes the configuration from the YAML file
func InitCfg() Config {
	// Parse the --cfg flag
	cfgFile := flag.String("cfg", "cfg.yaml", "Path to the configuration YAML file")
	flag.Parse()

	// Read the YAML file
	file, err := os.Open(*cfgFile)

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
