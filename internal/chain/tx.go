package chain

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

const COINBASE = "COINBASE"

// Transaction represents a transaction in the lab-chain network
type Transaction struct {
	From      string   `json:"from"`      // Sender's address
	To        string   `json:"to"`        // Recipient's address
	Amount    *big.Int `json:"amount"`    // Amount to transfer in lab-coins
	Nonce     uint64   `json:"nonce"`     // Transaction nonce
	Price     *big.Int `json:"price"`     // LC price in lab-coins
	Signature []byte   `json:"signature"` // Transaction signature
}

// CreateTx creates a new transaction with the given parameters and signs it
func CreateTx(fromPriv *ecdsa.PrivateKey, to string, amount, price *big.Int, chain *Chain) (*Transaction, error) {
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

// VerifySignature verifies the transaction's signature
func (tx *Transaction) VerifySignature() (bool, error) {
	if tx.From == COINBASE {
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

// PublishTx publishes a transaction to the specified pubsub topic
func (tx *Transaction) PublishTx(ctx context.Context, txTopic *pubsub.Topic) error {
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

// serializeTxs serializes the transactions into a byte slice
func serializeTxs(txs []*Transaction) []byte {
	var data []byte

	for _, tx := range txs {
		b, _ := json.Marshal(tx)
		data = append(data, b...)
	}

	return data
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
