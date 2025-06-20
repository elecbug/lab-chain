package chain

import (
	"sort"
	"sync"
)

// Mempool represents a memory pool for transactions
type Mempool struct {
	mu   sync.RWMutex
	pool map[string]*Transaction // key: tx hash or signature
}

// Sort sorts the transactions in the mempool by nonce
func (mp *Mempool) Sort() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var txs []*Transaction
	for _, tx := range mp.pool {
		txs = append(txs, tx)
	}

	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Nonce == txs[j].Nonce {
			return txs[i].From < txs[j].From // Secondary sort by sender address if nonces are equal
		}

		return txs[i].Nonce < txs[j].Nonce
	})

	mp.pool = make(map[string]*Transaction)
	for _, tx := range txs {
		mp.pool[string(tx.Signature)] = tx
	}
}

// NewMempool creates a new instance of Mempool
func NewMempool() *Mempool {
	return &Mempool{
		pool: make(map[string]*Transaction),
	}
}

// PickTopTxs returns the top count transactions from the mempool sorted by price,
// and removes them from the mempool.
func (mp *Mempool) PickTopTxs(count int) []*Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()

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
		txs = txs[:count]
	}

	// Remove selected transactions from the pool
	for _, tx := range txs {
		delete(mp.pool, string(tx.Signature))
	}

	return txs
}

// Remove deletes a transaction from the mempool by hash
func (mp *Mempool) Remove(tx *Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	delete(mp.pool, string(tx.Signature))
}
