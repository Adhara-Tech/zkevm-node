package sequencer

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/pool"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Worker represents the worker component of the sequencer
type Worker struct {
	pool             map[string]*addrQueue
	txSortedList     *txSortedList
	pendingToStore   map[common.Hash]*TxTracker
	workerMutex      sync.Mutex
	state            stateInterface
	batchConstraints state.BatchConstraintsCfg
	readyTxsCond     *timeoutCond
}

// NewWorker creates an init a worker
func NewWorker(state stateInterface, constraints state.BatchConstraintsCfg, readyTxsCond *timeoutCond) *Worker {
	w := Worker{
		pool:             make(map[string]*addrQueue),
		txSortedList:     newTxSortedList(),
		pendingToStore:   make(map[common.Hash]*TxTracker),
		state:            state,
		batchConstraints: constraints,
		readyTxsCond:     readyTxsCond,
	}

	return &w
}

// NewTxTracker creates and inits a TxTracker
func (w *Worker) NewTxTracker(tx types.Transaction, usedZKCounters state.ZKCounters, reservedZKCounters state.ZKCounters, ip string) (*TxTracker, error) {
	return newTxTracker(tx, usedZKCounters, reservedZKCounters, ip)
}

// AddTxTracker adds a new Tx to the Worker
func (w *Worker) AddTxTracker(ctx context.Context, tx *TxTracker) (replacedTx *TxTracker, dropReason error) {
	w.workerMutex.Lock()

	// Make sure the IP is valid.
	if tx.IP != "" && !pool.IsValidIP(tx.IP) {
		w.workerMutex.Unlock()
		return nil, pool.ErrInvalidIP
	}

	// Make sure the transaction's reserved ZKCounters are within the constraints.
	if !w.batchConstraints.IsWithinConstraints(tx.ReservedZKCounters) {
		log.Errorf("outOfCounters error (node level) for tx %s", tx.Hash.String())
		w.workerMutex.Unlock()
		return nil, pool.ErrOutOfCounters
	}

	addr, found := w.pool[tx.FromStr]
	if !found {
		// Unlock the worker to let execute other worker functions while creating the new AddrQueue
		w.workerMutex.Unlock()

		root, err := w.state.GetLastStateRoot(ctx, nil)
		if err != nil {
			dropReason = fmt.Errorf("error getting last state root from hashdb service, error: %v", err)
			log.Error(dropReason)
			return nil, dropReason
		}
		nonce, err := w.state.GetNonceByStateRoot(ctx, tx.From, root)
		if err != nil {
			dropReason = fmt.Errorf("error getting nonce for address %s from hashdb service, error: %v", tx.From, err)
			log.Error(dropReason)
			return nil, dropReason
		}
		balance, err := w.state.GetBalanceByStateRoot(ctx, tx.From, root)
		if err != nil {
			dropReason = fmt.Errorf("error getting balance for address %s from hashdb service, error: %v", tx.From, err)
			log.Error(dropReason)
			return nil, dropReason
		}

		addr = newAddrQueue(tx.From, nonce.Uint64(), balance)

		// Lock again the worker
		w.workerMutex.Lock()

		w.pool[tx.FromStr] = addr
		log.Debugf("new addrQueue %s created (nonce: %d, balance: %s)", tx.FromStr, nonce.Uint64(), balance.String())
	}

	// Add the txTracker to Addr and get the newReadyTx and prevReadyTx
	log.Infof("added new tx %s (nonce: %d, gasPrice: %d) to addrQueue %s (nonce: %d, balance: %d)", tx.HashStr, tx.Nonce, tx.GasPrice, addr.fromStr, addr.currentNonce, addr.currentBalance)
	var newReadyTx, prevReadyTx, repTx *TxTracker
	newReadyTx, prevReadyTx, repTx, dropReason = addr.addTx(tx)
	if dropReason != nil {
		log.Infof("dropped tx %s from addrQueue %s, reason: %s", tx.HashStr, tx.FromStr, dropReason.Error())
		w.workerMutex.Unlock()
		return repTx, dropReason
	}

	// Update the txSortedList (if needed)
	if prevReadyTx != nil {
		log.Debugf("prevReadyTx %s (nonce: %d, gasPrice: %d, addr: %s) deleted from TxSortedList", prevReadyTx.HashStr, prevReadyTx.Nonce, prevReadyTx.GasPrice, tx.FromStr)
		w.txSortedList.delete(prevReadyTx)
	}
	if newReadyTx != nil {
		log.Debugf("newReadyTx %s (nonce: %d, gasPrice: %d, addr: %s) added to TxSortedList", newReadyTx.HashStr, newReadyTx.Nonce, newReadyTx.GasPrice, tx.FromStr)
		w.addTxToSortedList(newReadyTx)
	}

	if repTx != nil {
		log.Debugf("tx %s (nonce: %d, gasPrice: %d, addr: %s) has been replaced", repTx.HashStr, repTx.Nonce, repTx.GasPrice, tx.FromStr)
	}

	w.workerMutex.Unlock()
	return repTx, nil
}

func (w *Worker) applyAddressUpdate(from common.Address, fromNonce *uint64, fromBalance *big.Int) (*TxTracker, *TxTracker, []*TxTracker) {
	addrQueue, found := w.pool[from.String()]

	if found {
		newReadyTx, prevReadyTx, txsToDelete := addrQueue.updateCurrentNonceBalance(fromNonce, fromBalance)

		// Update the TxSortedList (if needed)
		if prevReadyTx != nil {
			log.Debugf("prevReadyTx %s (nonce: %d, gasPrice: %d) deleted from TxSortedList", prevReadyTx.Hash.String(), prevReadyTx.Nonce, prevReadyTx.GasPrice)
			w.txSortedList.delete(prevReadyTx)
		}
		if newReadyTx != nil {
			log.Debugf("newReadyTx %s (nonce: %d, gasPrice: %d) added to TxSortedList", newReadyTx.Hash.String(), newReadyTx.Nonce, newReadyTx.GasPrice)
			w.addTxToSortedList(newReadyTx)
		}

		return newReadyTx, prevReadyTx, txsToDelete
	}

	return nil, nil, nil
}

// UpdateAfterSingleSuccessfulTxExecution updates the touched addresses after execute on Executor a successfully tx
func (w *Worker) UpdateAfterSingleSuccessfulTxExecution(from common.Address, touchedAddresses map[common.Address]*state.InfoReadWrite) []*TxTracker {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()
	if len(touchedAddresses) == 0 {
		log.Warnf("touchedAddresses is nil or empty")
	}
	txsToDelete := make([]*TxTracker, 0)
	touchedFrom, found := touchedAddresses[from]
	if found {
		fromNonce, fromBalance := touchedFrom.Nonce, touchedFrom.Balance
		_, _, txsToDelete = w.applyAddressUpdate(from, fromNonce, fromBalance)
	} else {
		log.Warnf("from address %s not found in touchedAddresses", from.String())
	}

	for addr, addressInfo := range touchedAddresses {
		if addr != from {
			_, _, txsToDeleteTemp := w.applyAddressUpdate(addr, nil, addressInfo.Balance)
			txsToDelete = append(txsToDelete, txsToDeleteTemp...)
		}
	}
	return txsToDelete
}

// MoveTxToNotReady move a tx to not ready after it fails to execute
func (w *Worker) MoveTxToNotReady(txHash common.Hash, from common.Address, actualNonce *uint64, actualBalance *big.Int) []*TxTracker {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()
	log.Debugf("move tx %s to notReady (from: %s, actualNonce: %d, actualBalance: %s)", txHash.String(), from.String(), actualNonce, actualBalance.String())

	addrQueue, found := w.pool[from.String()]
	if found {
		// Sanity check. The txHash must be the readyTx
		if addrQueue.readyTx == nil || txHash.String() != addrQueue.readyTx.HashStr {
			readyHashStr := ""
			if addrQueue.readyTx != nil {
				readyHashStr = addrQueue.readyTx.HashStr
			}
			log.Warnf("tx %s is not the readyTx %s", txHash.String(), readyHashStr)
		}
	}
	_, _, txsToDelete := w.applyAddressUpdate(from, actualNonce, actualBalance)

	return txsToDelete
}

// deleteTx deletes a regular tx from the addrQueue
func (w *Worker) deleteTx(txHash common.Hash, addr common.Address) *TxTracker {
	addrQueue, found := w.pool[addr.String()]
	if found {
		deletedTx, isReady := addrQueue.deleteTx(txHash)
		if deletedTx != nil {
			if isReady {
				log.Debugf("tx %s deleted from TxSortedList", deletedTx.Hash)
				w.txSortedList.delete(deletedTx)
			}
		} else {
			log.Warnf("tx %s not found in addrQueue %s", txHash, addr)
		}

		return deletedTx
	} else {
		log.Warnf("addrQueue %s not found", addr)

		return nil
	}
}

// DeleteTx deletes a regular tx from the addrQueue
func (w *Worker) DeleteTx(txHash common.Hash, addr common.Address) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	w.deleteTx(txHash, addr)
}

// DeleteForcedTx deletes a forced tx from the addrQueue
func (w *Worker) DeleteForcedTx(txHash common.Hash, addr common.Address) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	addrQueue, found := w.pool[addr.String()]
	if found {
		addrQueue.deleteForcedTx(txHash)
	} else {
		log.Warnf("addrQueue %s not found", addr.String())
	}
}

// UpdateTxZKCounters updates the ZKCounter of a tx
func (w *Worker) UpdateTxZKCounters(txHash common.Hash, addr common.Address, usedZKCounters state.ZKCounters, reservedZKCounters state.ZKCounters) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	log.Infof("update ZK counters for tx %s addr %s", txHash.String(), addr.String())
	// TODO: log in a single line, log also reserved resources
	log.Debugf("counters.CumulativeGasUsed: %d", usedZKCounters.GasUsed)
	log.Debugf("counters.UsedKeccakHashes: %d", usedZKCounters.KeccakHashes)
	log.Debugf("counters.UsedPoseidonHashes: %d", usedZKCounters.PoseidonHashes)
	log.Debugf("counters.UsedPoseidonPaddings: %d", usedZKCounters.PoseidonPaddings)
	log.Debugf("counters.UsedMemAligns: %d", usedZKCounters.MemAligns)
	log.Debugf("counters.UsedArithmetics: %d", usedZKCounters.Arithmetics)
	log.Debugf("counters.UsedBinaries: %d", usedZKCounters.Binaries)
	log.Debugf("counters.UsedSteps: %d", usedZKCounters.Steps)
	log.Debugf("counters.UsedSha256Hashes_V2: %d", usedZKCounters.Sha256Hashes_V2)

	addrQueue, found := w.pool[addr.String()]

	if found {
		addrQueue.UpdateTxZKCounters(txHash, usedZKCounters, reservedZKCounters)
	} else {
		log.Warnf("addrQueue %s not found", addr.String())
	}
}

// MoveTxPendingToStore moves a tx to pending to store list
func (w *Worker) MoveTxPendingToStore(txHash common.Hash, addr common.Address) {
	// TODO: Add test for this function

	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	// Delete from worker pool and addrQueue
	deletedTx := w.deleteTx(txHash, addr)

	// Add tx to pending to store list in worker
	if deletedTx != nil {
		w.pendingToStore[txHash] = deletedTx
	} else {
		log.Warnf("tx %s not found when moving it to pending to store, address: %s", txHash, addr)
	}

	// Add tx to pending to store list in addrQueue
	if addrQueue, found := w.pool[addr.String()]; found {
		addrQueue.addPendingTxToStore(txHash)
	} else {
		log.Warnf("addrQueue %s not found when moving tx %s to pending to store", addr, txHash)
	}
}

// RestoreTxsPendingToStore restores the txs pending to store and move them to the worker pool to be processed again
func (w *Worker) RestoreTxsPendingToStore(ctx context.Context) ([]*TxTracker, []*TxTracker) {
	// TODO: Add test for this function
	// TODO: We need to process restored txs in the same order we processed initially

	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	addrList := make(map[common.Address]struct{})
	txsList := []*TxTracker{}

	// Add txs pending to store to the list that will include all the txs to reprocess again
	// and we get also the addresses of theses txs since we will need to recreate them
	for _, tx := range w.pendingToStore {
		txsList = append(txsList, tx)
		addrList[tx.From] = struct{}{}
	}

	// Add txs from addrQueues that will be recreated and delete addrQueues from the pool list
	for addr := range addrList {
		addrQueue, found := w.pool[addr.String()]
		if found {
			txsList = append(txsList, addrQueue.getTransactions()...)
			delete(w.pool, addr.String())
		}
	}

	// Clear pendingToStore list
	clear(w.pendingToStore)

	replacedTxs := []*TxTracker{}
	droppedTxs := []*TxTracker{}
	// Add again in the worker the txs to restore (this will recreate addrQueue)
	for _, restoredTx := range txsList {
		replacedTx, dropReason := w.AddTxTracker(ctx, restoredTx)
		if dropReason != nil {
			droppedTxs = append(droppedTxs, restoredTx)
		}
		if replacedTx != nil {
			droppedTxs = append(replacedTxs, restoredTx)
		}
	}

	// In this scenario we shouldn't have dropped or replaced txs but we return it just in case
	return droppedTxs, replacedTxs
}

// AddForcedTx adds a forced tx to the addrQueue
func (w *Worker) AddForcedTx(txHash common.Hash, addr common.Address) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	if addrQueue, found := w.pool[addr.String()]; found {
		addrQueue.addForcedTx(txHash)
	} else {
		log.Warnf("addrQueue %s not found", addr.String())
	}
}

// DeleteTxPendingToStore delete a tx from the addrQueue list of pending txs to store in the DB (trusted state)
func (w *Worker) DeleteTxPendingToStore(txHash common.Hash, addr common.Address) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	// Delete tx from pending to store list in worker
	if _, found := w.pendingToStore[txHash]; found {
		delete(w.pendingToStore, txHash)
	} else {
		log.Warnf("tx %s not found when deleting it from worker pool", txHash)
	}

	// Delete tx from pending to store list in addrQueue
	if addrQueue, found := w.pool[addr.String()]; found {
		addrQueue.deletePendingTxToStore(txHash)
	} else {
		log.Warnf("addrQueue %s not found when deleting pending to store tx %s", addr, txHash)
	}
}

// GetBestFittingTx gets the most efficient tx that fits in the available batch resources
func (w *Worker) GetBestFittingTx(resources state.BatchResources) (*TxTracker, error) {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	if w.txSortedList.len() == 0 {
		return nil, ErrTransactionsListEmpty
	}

	var (
		tx         *TxTracker
		foundMutex sync.RWMutex
	)

	nGoRoutines := runtime.NumCPU()
	foundAt := -1

	wg := sync.WaitGroup{}
	wg.Add(nGoRoutines)

	// Each go routine looks for a fitting tx
	for i := 0; i < nGoRoutines; i++ {
		go func(n int, bresources state.BatchResources) {
			defer wg.Done()
			for i := n; i < w.txSortedList.len(); i += nGoRoutines {
				foundMutex.RLock()
				if foundAt != -1 && i > foundAt {
					foundMutex.RUnlock()
					return
				}
				foundMutex.RUnlock()

				txCandidate := w.txSortedList.getByIndex(i)
				overflow, _ := bresources.Sub(state.BatchResources{ZKCounters: txCandidate.ReservedZKCounters, Bytes: txCandidate.Bytes})
				if overflow {
					// We don't add this Tx
					continue
				}

				foundMutex.Lock()
				if foundAt == -1 || foundAt > i {
					foundAt = i
					tx = txCandidate
				}
				foundMutex.Unlock()

				return
			}
		}(i, resources)
	}
	wg.Wait()

	if foundAt != -1 {
		log.Debugf("best fitting tx %s found at index %d with gasPrice %d", tx.HashStr, foundAt, tx.GasPrice)
		return tx, nil
	} else {
		return nil, ErrNoFittingTransaction
	}
}

// ExpireTransactions deletes old txs
func (w *Worker) ExpireTransactions(maxTime time.Duration) []*TxTracker {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()

	var txs []*TxTracker

	log.Debugf("expire transactions started, addrQueue length: %d", len(w.pool))
	for _, addrQueue := range w.pool {
		subTxs, prevReadyTx := addrQueue.ExpireTransactions(maxTime)
		txs = append(txs, subTxs...)

		if prevReadyTx != nil {
			w.txSortedList.delete(prevReadyTx)
		}

		/*if addrQueue.IsEmpty() {
			delete(w.pool, addrQueue.fromStr)
		}*/
	}
	log.Debugf("expire transactions ended, addrQueue length: %d, delete count: %d ", len(w.pool), len(txs))

	return txs
}

func (w *Worker) addTxToSortedList(readyTx *TxTracker) {
	w.txSortedList.add(readyTx)
	if w.txSortedList.len() == 1 {
		// The txSortedList was empty before to add the new tx, we notify finalizer that we have new ready txs to process
		w.readyTxsCond.L.Lock()
		w.readyTxsCond.Signal()
		w.readyTxsCond.L.Unlock()
	}
}
