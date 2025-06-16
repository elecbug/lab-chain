package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// GenerateMasterKey generates a BIP-44 master key and returns it
func GenerateMasterKey() (*bip32.Key, error) {
	log := logger.AppLogger

	log.Infof("generating BIP-44 mnemonic")

	entropy, err := bip39.NewEntropy(128)

	if err != nil {
		return nil, fmt.Errorf("failed to generate entropy: %v", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)

	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %v", err)
	} else {
		log.Infof("generated mnemonic: %s", mnemonic)
	}

	seed := bip39.NewSeed(mnemonic, "")

	masterKey, err := bip32.NewMasterKey(seed)

	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %v", err)
	} else {
		log.Infof("master key created successfully")
	}

	return masterKey, nil
}

// SaveMasterKey saves the master key to a file
func SaveMasterKey(file string, masterKey *bip32.Key) error {
	log := logger.AppLogger

	serialized := masterKey.String()

	if err := os.WriteFile(fmt.Sprintf("%s.master", file), []byte(serialized), 0600); err != nil {
		return fmt.Errorf("failed to save master key: %v", err)
	} else {
		log.Infof("master key saved to file successfully")
	}

	return nil
}

// LoadMasterKey loads the master key from a file
func LoadMasterKey(file string) (*bip32.Key, error) {
	log := logger.AppLogger

	data, err := os.ReadFile(fmt.Sprintf("%s.master", file))

	if err != nil {
		return nil, fmt.Errorf("failed to read master key file: %v", err)
	}

	loadedKey, err := bip32.B58Deserialize(string(data))

	if err != nil {
		return nil, fmt.Errorf("failed to deserialize master key: %v", err)
	} else {
		log.Infof("master key loaded successfully")
	}

	return loadedKey, nil
}

// GenerateWallet generates a BIP-44 key pair for the specified index
func GenerateWallet(masterKey *bip32.Key, index int) (*ecdsa.PrivateKey, *common.Address, error) {
	log := logger.AppLogger

	purpose, _ := masterKey.NewChildKey(44 + bip32.FirstHardenedChild)
	coinType, _ := purpose.NewChildKey(60 + bip32.FirstHardenedChild)
	account, _ := coinType.NewChildKey(0 + bip32.FirstHardenedChild)
	change, _ := account.NewChildKey(0)
	addressKey, _ := change.NewChildKey(uint32(index))

	privateKey, err := crypto.ToECDSA(addressKey.Key)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert to ECDSA: %v", err)
	} else {
		log.Infof("private key generated successfully")
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKey)

	return privateKey, &address, nil
}
