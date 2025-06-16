package blockchain

import (
	"sort"
	"sync"
)

// Mempool represents a memory pool for transactions
type Mempool struct {
	mu   sync.RWMutex
	pool map[string]*Transaction // key: tx hash or signature
}

// NewMempool creates a new instance of Mempool
func NewMempool() *Mempool {
	return &Mempool{
		pool: make(map[string]*Transaction),
	}
}

// PickTopTxs returns the top count transactions from the mempool sorted by price
func (mp *Mempool) PickTopTxs(count int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Copy to slice
	var txs []*Transaction
	for _, tx := range mp.pool {
		txs = append(txs, tx)
	}

	// Sort by price descending
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Price.Cmp(txs[j].Price) > 0
	})

	if len(txs) > count {
		return txs[:count]
	}
	return txs
}
