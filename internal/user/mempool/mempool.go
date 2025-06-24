package mempool

import (
	"sort"
	"sync"

	"github.com/elecbug/lab-chain/internal/chain/tx"
)

// Mempool represents a memory pool for transactions
type Mempool struct {
	Mu   sync.RWMutex
	pool map[string]*tx.Transaction // key: tx hash or signature
}

// Add adds a transaction to the mempool if it does not already exist
func (mp *Mempool) Add(txID string, t *tx.Transaction) bool {

	if _, exists := mp.pool[txID]; !exists {
		mp.pool[txID] = t

		return true
	} else {
		return false
	}
}

// GetBase returns the base count of transactions for a given address
func (mp *Mempool) GetBase(addr string) int {
	mp.Mu.RLock()
	defer mp.Mu.RUnlock()

	base := 0
	for _, tx := range mp.pool {
		if tx.From == addr {
			base++
		}
	}

	return base
}

// Sort sorts the transactions in the mempool by nonce
func (mp *Mempool) Sort() {
	mp.Mu.Lock()
	defer mp.Mu.Unlock()

	var txs []*tx.Transaction
	for _, tx := range mp.pool {
		txs = append(txs, tx)
	}

	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Nonce == txs[j].Nonce {
			return txs[i].From < txs[j].From // Secondary sort by sender address if nonces are equal
		}

		return txs[i].Nonce < txs[j].Nonce
	})

	mp.pool = make(map[string]*tx.Transaction)
	for _, tx := range txs {
		mp.pool[string(tx.Signature)] = tx
	}
}

// NewMempool creates a new instance of Mempool
func NewMempool() *Mempool {
	return &Mempool{
		pool: make(map[string]*tx.Transaction),
	}
}

// PickTopTxs returns the top count transactions from the mempool sorted by price,
// and removes them from the mempool.
func (mp *Mempool) PickTopTxs(count int) []*tx.Transaction {
	mp.Mu.Lock()
	defer mp.Mu.Unlock()

	// Copy to slice
	var txs []*tx.Transaction
	for _, tx := range mp.pool {
		txs = append(txs, tx)
	}

	// Sort by price descending
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Price.Cmp(txs[j].Price) > 0
	})

	if len(txs) > count {
		txs = txs[:count]
	}

	// Remove selected transactions from the pool
	for _, tx := range txs {
		delete(mp.pool, string(tx.Signature))
	}

	return txs
}

// Remove deletes a transaction from the mempool by hash
func (mp *Mempool) Remove(tx *tx.Transaction) {
	mp.Mu.Lock()
	defer mp.Mu.Unlock()

	delete(mp.pool, string(tx.Signature))
}
