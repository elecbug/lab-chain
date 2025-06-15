package cfg

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
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

// InitSetting initializes the configuration from the YAML file
func InitSetting() (*Config, *crypto.PrivKey, error) {
	cfgFile := flag.String("cfg", "cfg.yaml", "Path to the configuration file")
	keyFile := flag.String("key", "keypair", "Path to the key file name without extention (optional)")
	flag.Parse()

	config, err := setConfig(*cfgFile)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to set configuration: %v", err)
	}

	key, err := setKeyPair(*keyFile)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to set key pair: %v", err)
	}

	return config, key, nil
}

// GetConfig returns the configuration from the YAML file
func setConfig(cfgFile string) (*Config, error) {
	file, err := os.Open(cfgFile)

	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %v", err)
	}
	defer file.Close()

	var config Config

	decoder := yaml.NewDecoder(file)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML file into cfg.Config: %v", err)
	}

	log.Printf("Configuration: %+v", config)

	return &config, nil
}

// setKeyPair checks for existing key files and generates a new key pair if they do not exist
func setKeyPair(file string) (*crypto.PrivKey, error) {
	priv := fmt.Sprintf("%s.pem", file)
	pub := fmt.Sprintf("%s.pub", file)

	_, privErr := os.Stat(priv)
	_, pubErr := os.Stat(pub)

	if os.IsNotExist(privErr) || os.IsNotExist(pubErr) {
		log.Printf("Key files %s or %s do not exist, generating new key pair...", priv, pub)
		privKey, pubKey, err := crypto.GenerateEd25519Key(nil)

		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %v", err)
		}

		// Save the private key
		if bs, err := crypto.MarshalPrivateKey(privKey); err != nil {
			return nil, fmt.Errorf("failed to write private key to file %s: %v", priv, err)
		} else {
			if err := os.WriteFile(priv, bs, 0600); err != nil {
				return nil, fmt.Errorf("failed to write private key to file %s: %v", priv, err)
			}
		}

		// Save the public key
		if bs, err := crypto.MarshalPublicKey(pubKey); err != nil {
			return nil, fmt.Errorf("failed to write public key to file %s: %v", pub, err)
		} else {
			if err := os.WriteFile(pub, bs, 0644); err != nil {
				return nil, fmt.Errorf("failed to write public key to file %s: %v", pub, err)
			}
		}

		log.Printf("New key pair generated and saved to %s and %s", priv, pub)

		return &privKey, nil
	} else {
		log.Printf("Key files %s and %s already exist, loading existing key pair...", priv, pub)

		// Load the private key
		privKeyBytes, err := os.ReadFile(priv)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key from file %s: %v", priv, err)
		}

		privKey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal private key from file %s: %v", priv, err)
		}

		// Load the public key
		pubKeyBytes, err := os.ReadFile(pub)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key from file %s: %v", pub, err)
		}

		_, err = crypto.UnmarshalPublicKey(pubKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal public key from file %s: %v", pub, err)
		}

		log.Printf("Existing key pair loaded from %s and %s", priv, pub)

		return &privKey, nil
	}
}
