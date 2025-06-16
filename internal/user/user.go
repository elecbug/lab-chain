package user

import (
	"context"
	"crypto/ecdsa"

	"github.com/elecbug/lab-chain/internal/block"
	"github.com/elecbug/lab-chain/internal/transaction"
	"github.com/ethereum/go-ethereum/common"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/tyler-smith/go-bip32"
)

type User struct {
	Context        context.Context // Context for user operations
	MasterKey      *bip32.Key      // BIP-44 master key
	CurrentPrivKey *ecdsa.PrivateKey
	CurrentAddress *common.Address
	Blockchain     *block.Blockchain    // Reference to the blockchain
	TxTopic        *pubsub.Topic        // Pubsub topic for transactions
	BlockTopic     *pubsub.Topic        // Pubsub topic for blocks
	MemPool        *transaction.Mempool // Memory pool for transactions
}
