package blockchain

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Transaction represents a transaction in the lab-chain network
type Transaction struct {
	From      string   `json:"from"`      // Sender's address
	To        string   `json:"to"`        // Recipient's address
	Amount    *big.Int `json:"amount"`    // Amount to transfer in lab-coins
	Nonce     uint64   `json:"nonce"`     // Transaction nonce
	Price     *big.Int `json:"price"`     // LC price in lab-coins
	Signature []byte   `json:"signature"` // Transaction signature
}

// VerifySignature verifies the transaction's signature
func (tx *Transaction) VerifySignature() (bool, error) {
	if tx.From == "COINBASE" {
		// Coinbase transactions do not have a signature
		return true, nil
	}

	hash := tx.hash()
	sig := tx.Signature

	if len(sig) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}

	pubKey, err := crypto.SigToPub(hash, sig)

	if err != nil {
		return false, fmt.Errorf("failed to recover public key from signature: %v", err)
	}

	derivedAddr := crypto.PubkeyToAddress(*pubKey)

	return strings.EqualFold(derivedAddr.Hex(), tx.From), nil
}

// CreateTx creates a new transaction with the given parameters and signs it
func CreateTx(fromPriv *ecdsa.PrivateKey, to string, amount, price *big.Int, chain *Blockchain) (*Transaction, error) {
	log := logger.LabChainLogger

	pubKey := fromPriv.Public().(*ecdsa.PublicKey)
	fromAddr := crypto.PubkeyToAddress(*pubKey)

	tx := &Transaction{
		From:   fromAddr.Hex(),
		To:     to,
		Amount: amount,
		Nonce:  chain.GetNonce(fromAddr.Hex()),
		Price:  price,
	}

	err := tx.sign(fromPriv)

	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	} else {
		log.Infof("transaction signed successfully: %s -> %s, amount: %s, price: %s, nonce: %d",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String(), tx.Nonce)
	}

	return tx, nil
}

// PublishTx publishes a transaction to the specified pubsub topic
func PublishTx(ctx context.Context, txTopic *pubsub.Topic, tx *Transaction) error {
	log := logger.LabChainLogger

	txBs, err := serializeTx(tx)

	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %v", err)
	} else {
		log.Infof("transaction serialized successfully: %s -> %s, amount: %s, price: %s, nonce: %d",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String(), tx.Nonce)
	}

	err = txTopic.Publish(ctx, txBs)

	if err != nil {
		return fmt.Errorf("failed to publish transaction: %v", err)
	} else {
		log.Infof("transaction published successfully: %s -> %s, amount: %s, price: %s, nonce: %d",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String(), tx.Nonce)
	}

	return nil
}

// RunSubscribeAndCollectTx listens for incoming transactions on the pubsub subscription
func RunSubscribeAndCollectTx(ctx context.Context, sub *pubsub.Subscription, mempool *Mempool, chain *Blockchain) {
	log := logger.LabChainLogger

	go func() {
		for {
			msg, err := sub.Next(ctx)

			if err != nil {
				log.Errorf("failed to receive pubsub message: %v", err)
				continue
			}

			tx, err := deserializeTx(msg.Data)
			if err != nil {
				log.Warnf("invalid tx: failed to deserialize: %v", err)
				continue
			}

			ok, err := tx.VerifySignature()
			if err != nil || !ok {
				log.Warnf("invalid tx: signature verification failed: %v", err)
				continue
			}

			if chain != nil {
				required := new(big.Int).Add(tx.Amount, tx.Price)
				balance := chain.GetBalance(tx.From)
				if balance.Cmp(required) < 0 {
					log.Warnf("invalid tx: insufficient balance. required: %s, actual: %s", required.String(), balance.String())
					continue
				}
			}

			txID := string(tx.Signature)
			mempool.mu.Lock()
			if _, exists := mempool.pool[txID]; !exists {
				mempool.pool[txID] = tx
				log.Infof("transaction received and stored: %s -> %s, amount: %s", tx.From, tx.To, tx.Amount.String())
			} else {
				log.Debugf("transaction already in mempool, skipping: %s", txID)
			}
			mempool.mu.Unlock()
		}
	}()
}

// hash computes the hash of the transaction for signing and verification
func (tx *Transaction) hash() []byte {
	// Create a clone of the transaction without the signature for hashing
	clone := *tx
	clone.Signature = nil

	jsonBytes, _ := json.Marshal(clone)
	hash := crypto.Keccak256(jsonBytes)

	return hash
}

// NewTransaction creates a new transaction with the given parameters
func (tx *Transaction) sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.hash()
	sig, err := crypto.Sign(hash, privKey)

	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	tx.Signature = sig

	return nil
}

// serializeTx and deserialize functions for transaction
func serializeTx(tx *Transaction) ([]byte, error) {
	jsonBytes, err := json.Marshal(tx)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %v", err)
	}

	return jsonBytes, nil
}

// deserializeTx converts JSON bytes back into a Transaction object
func deserializeTx(data []byte) (*Transaction, error) {
	var tx Transaction

	err := json.Unmarshal(data, &tx)

	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return &tx, nil
}
