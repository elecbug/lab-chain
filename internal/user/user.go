package user

import (
	"context"
	"crypto/ecdsa"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/ethereum/go-ethereum/common"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/tyler-smith/go-bip32"
)

type User struct {
	Context        context.Context // Context for user operations
	MasterKey      *bip32.Key      // BIP-44 master key
	CurrentPrivKey *ecdsa.PrivateKey
	CurrentAddress *common.Address
	Chain          *chain.Chain   // Reference to the blockchain
	TxTopic        *pubsub.Topic  // Pubsub topic for transactions
	BlockTopic     *pubsub.Topic  // Pubsub topic for blocks
	MemPool        *chain.Mempool // Memory pool for transactions
	PeerID         peer.ID        // Peer ID of the user in the network
}
