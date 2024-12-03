package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygonHermez/zkevm-node/config"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/0xPolygonHermez/zkevm-node/test/constants"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// dockersArePreLaunched is a flag that indicates if dockers are pre-launched, used for local development
	// avoiding launch time and reset database time at end (so you can check the database after the test)
	dockersArePreLaunched = false
	gerFinalityBlocks     = uint64(9223372036854775807) // The biggeset uint64
)

const (
	invalidParamsErrorCode = -32602
	toAddressHex           = "0x4d5Cf5032B2a844602278b01199ED191A86c93ff"
	forkID6                = 6
)

type l1Stuff struct {
	ethClient       *ethclient.Client
	authSequencer   *bind.TransactOpts
	authForcedBatch *bind.TransactOpts
	zkEvmAddr       common.Address
	zkEvm           *etrogpolygonzkevm.Etrogpolygonzkevm
}

type l2Stuff struct {
	opsman        *operations.Manager
	authSequencer *bind.TransactOpts
	client        *ethclient.Client
	amount        *big.Int
	gasLimit      uint64
	gasPrice      *big.Int
	nonce         uint64
}

func main() {
	log.Infof("Running ForcedBatches ==========================")
	//nTxs := 10
	ctx := context.Background()
	l2 := setupEnvironment(ctx)
	l1 := setupEnvironmentL1(ctx)
	//l2BlockNumbersTxsBeforeForcedBatch := generateTxsBeforeSendingForcedBatch(ctx, nTxs, l2)
	time.Sleep(2 * time.Second)
	l2.amount = big.NewInt(0).Add(l2.amount, big.NewInt(10))
	encodedTxs := generateSignedAndEncodedTxForForcedBatch(ctx, l2)
	forcedBatch, err := sendForcedBatch(ctx, encodedTxs, l2.opsman, l1)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(forcedBatch)
	//checkThatPreviousTxsWereProcessedWithinPreviousClosedBatch(ctx, l2.opsman.State(), l2BlockNumbersTxsBeforeForcedBatch, forcedBatch.BatchNumber)
}

func generateTxsBeforeSendingForcedBatch(ctx context.Context, nTxs int, l2 *l2Stuff) []*big.Int {
	toAddress := common.HexToAddress(toAddressHex)
	txs := make([]*types.Transaction, 0, nTxs)
	for i := 0; i < nTxs; i++ {
		tx := types.NewTransaction(l2.nonce, toAddress, l2.amount, l2.gasLimit, l2.gasPrice, nil)
		l2.nonce = l2.nonce + 1
		txs = append(txs, tx)
	}

	var l2BlockNumbers []*big.Int
	l2BlockNumbers, err := operations.ApplyL2Txs(ctx, txs, l2.authSequencer, l2.client, operations.VerifiedConfirmationLevel)
	if err != nil {
		log.Fatal(err)
	}
	return l2BlockNumbers
}

func checkThatPreviousTxsWereProcessedWithinPreviousClosedBatch(ctx context.Context, state *state.State, l2BlockNumbers []*big.Int, forcedBatchNumber uint64) {
	// Checking if all txs sent before the forced batch were processed within previous closed batch
	for _, l2blockNum := range l2BlockNumbers {
		batch, err := state.GetBatchByL2BlockNumber(ctx, l2blockNum.Uint64(), nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Info(batch.BatchNumber)
		log.Info(forcedBatchNumber)
	}
}

func generateSignedAndEncodedTxForForcedBatch(ctx context.Context, l2 *l2Stuff) []byte {
	toAddress := common.HexToAddress(toAddressHex)
	unsignedTx := types.NewTransaction(l2.nonce, toAddress, l2.amount, l2.gasLimit, l2.gasPrice, nil)
	signedTx, err := l2.authSequencer.Signer(l2.authSequencer.From, unsignedTx)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Forced Batch: 1 tx -> ", signedTx.Hash())
	encodedTxs, err := state.EncodeTransactions([]types.Transaction{*signedTx}, constants.EffectivePercentage, forkID6)
	if err != nil {
		log.Fatal(err)
	}
	return encodedTxs
}

func setupEnvironment(ctx context.Context) *l2Stuff {
	if !dockersArePreLaunched {
		err := operations.Teardown()
		if err != nil {
			log.Fatal(err)
		}
	}
	opsCfg := operations.GetDefaultOperationsConfig()
	opsCfg.State.MaxCumulativeGasUsed = 80000000000

	var opsman *operations.Manager
	var err error

	if !dockersArePreLaunched {
		log.Info("Launching dockers and resetting Database")
		opsman, err = operations.NewManager(ctx, opsCfg)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Setting Genesis")
		setInitialState(opsman)
	} else {
		log.Info("Using pre-launched dockers: no reset Database")
		opsman, err = operations.NewManagerNoInitDB(ctx, opsCfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Load account with balance on local genesis
	auth, err := operations.GetAuth(operations.DefaultSequencerPrivateKey, operations.DefaultL2ChainID)
	if err != nil {
		log.Fatal(err)
	}
	// Load eth client
	client, err := ethclient.Dial(operations.DefaultL2NetworkURL)
	if err != nil {
		log.Fatal(err)
	}
	// Send txs
	amount := big.NewInt(10000)
	senderBalance, err := client.BalanceAt(ctx, auth.From, nil)
	if err != nil {
		log.Fatal(err)
	}
	senderNonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress(toAddressHex)
	log.Infof("Receiver Addr: %v", toAddress.String())
	log.Infof("Sender Addr: %v", auth.From.String())
	log.Infof("Sender Balance: %v", senderBalance.String())
	log.Infof("Sender Nonce: %v", senderNonce)

	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: auth.From, To: &toAddress, Value: amount})
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return &l2Stuff{opsman, auth, client, amount, gasLimit, gasPrice, senderNonce}
}

func setupEnvironmentL1(ctx context.Context) *l1Stuff {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(operations.DefaultL1NetworkURL)
	if err != nil {
		log.Fatal(err)
	}
	authSequencer, err := operations.GetAuth(operations.DefaultSequencerPrivateKey, operations.DefaultL1ChainID)
	if err != nil {
		log.Fatal(err)
	}
	authForcedBatch, err := operations.GetAuth(operations.DefaultForcedBatchesPrivateKey, operations.DefaultL1ChainID)
	if err != nil {
		log.Fatal(err)
	}
	polSmc, err := pol.NewPol(common.HexToAddress(operations.DefaultL1PolSmartContract), ethClient)
	if err != nil {
		log.Fatal(err)
	}
	polAmount, _ := big.NewInt(0).SetString("9999999999999999999999", 0)
	log.Debugf("Charging pol from sequencer -> forcedBatchesAddress")
	txValue, err := polSmc.Transfer(authSequencer, common.HexToAddress(operations.DefaultForcedBatchesAddress), polAmount)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("Waiting for tx %s to be mined (transfer of pol from sequencer -> forcedBatches)", txValue.Hash().String())
	err = operations.WaitTxToBeMined(ctx, ethClient, txValue, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}
	balance, err := polSmc.BalanceOf(&bind.CallOpts{Pending: false}, common.HexToAddress(operations.DefaultSequencerAddress))
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("Account (sequencer) %s pol balance %s", operations.DefaultSequencerAddress, balance.String())

	balance, err = polSmc.BalanceOf(&bind.CallOpts{Pending: false}, common.HexToAddress(operations.DefaultForcedBatchesAddress))
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("Account (force_batches) %s pol balance %s", operations.DefaultForcedBatchesAddress, balance.String())
	log.Debugf("Approve to zkEVM SMC to spend %s pol", polAmount.String())
	_, err = polSmc.Approve(authForcedBatch, common.HexToAddress(operations.DefaultL1ZkEVMSmartContract), polAmount)
	if err != nil {
		log.Fatal(err)
	}

	zkEvmAddr := common.HexToAddress(operations.DefaultL1ZkEVMSmartContract)
	zkEvm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(zkEvmAddr, ethClient)
	if err != nil {
		log.Fatal(err)
	}
	return &l1Stuff{ethClient: ethClient, authSequencer: authSequencer, authForcedBatch: authForcedBatch, zkEvmAddr: zkEvmAddr, zkEvm: zkEvm}
}

func setInitialState(opsman *operations.Manager) {
	genesisFileAsStr, err := config.LoadGenesisFileAsString("../../config/test.genesis.config.json")
	if err != nil {
		log.Fatal(err)
	}
	genesisConfig, err := config.LoadGenesisFromJSONString(genesisFileAsStr)
	if err != nil {
		log.Fatal(err)
	}
	err = opsman.SetForkID(genesisConfig.Genesis.BlockNumber, forkID6)
	if err != nil {
		log.Fatal(err)
	}
	err = opsman.Setup()
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(5 * time.Second)
}

func sendForcedBatch(ctx context.Context, txs []byte, opsman *operations.Manager, l1 *l1Stuff) (*state.Batch, error) {
	st := opsman.State()

	initialGer, _, err := st.GetLatestGer(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Using address: ", l1.authForcedBatch.From)

	num, err := l1.zkEvm.LastForceBatch(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Number of forceBatches in the smc: ", num)

	rollupManagerAddr := common.HexToAddress(operations.DefaultL1RollupManagerSmartContract)
	rollupManager, err := etrogpolygonrollupmanager.NewEtrogpolygonrollupmanager(rollupManagerAddr, l1.ethClient)
	if err != nil {
		log.Fatal(err)
	}

	// Get tip
	tip, err := rollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Foced Batch Fee:%s", tip.String())
	managerAddress, err := l1.zkEvm.GlobalExitRootManager(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}

	manager, err := etrogpolygonzkevmglobalexitroot.NewEtrogpolygonzkevmglobalexitroot(managerAddress, l1.ethClient)
	if err != nil {
		log.Fatal(err)
	}

	rootInContract, err := manager.GetLastGlobalExitRoot(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	rootInContractHash := common.BytesToHash(rootInContract[:])

	log.Infof("Activating forced batches...")
	tx, err := l1.zkEvm.SetForceBatchAddress(l1.authSequencer, common.Address{})
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Forced batch is disallowed. Activated. Waiting for tx %s to be mined", tx.Hash())
	err = operations.WaitTxToBeMined(ctx, l1.ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	currentBlock, err := l1.ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("L1: currentBlock: number:%s Time():%s ", currentBlock.Number().String(), currentBlock.Time())

	// Send forceBatch
	tx, err = l1.zkEvm.ForceBatch(l1.authForcedBatch, txs, tip)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("TxHash: ", tx.Hash())
	time.Sleep(1 * time.Second)

	err = operations.WaitTxToBeMined(ctx, l1.ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	fb, vLog, err := findForcedBatchInL1Logs(ctx, currentBlock.Number(), l1)
	if err != nil {
		log.Errorf("failed to parse force batch log event, err: ", err)
	}
	ger := fb.LastGlobalExitRoot

	log.Debugf("log decoded: %+v", fb)
	log.Info("GlobalExitRoot: ", ger)
	log.Info("Transactions: ", common.Bytes2Hex(fb.Transactions))
	log.Info("ForcedBatchNum: ", fb.ForceBatchNum)
	fullBlock, err := l1.ethClient.BlockByHash(ctx, vLog.BlockHash)
	if err != nil {
		log.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		return nil, err
	}
	log.Info("MinForcedTimestamp: ", fullBlock.Time())
	//forcedBatch, err := st.GetBatchByForcedBatchNum(ctx, fb.ForceBatchNum, nil)
	//for err == state.ErrStateNotSynchronized {
	//	log.Infof("state not synced, waiting...")
	//	time.Sleep(1 * time.Second)
	//	forcedBatch, err = st.GetBatchByForcedBatchNum(ctx, fb.ForceBatchNum, nil)
	//}
	//
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Info(forcedBatch)
	//
	//log.Info("Waiting for batch to be virtualized...")
	//err = operations.WaitBatchToBeVirtualized(forcedBatch.BatchNumber, 4*time.Minute, st)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//log.Info("Waiting for batch to be consolidated...")
	//err = operations.WaitBatchToBeConsolidated(forcedBatch.BatchNumber, 4*time.Minute, st)
	//if err != nil {
	//	log.Fatal(err)
	//}

	if rootInContractHash != initialGer.GlobalExitRoot {
		log.Info("Checking if global exit root is updated...")
		finalGer, _, err := st.GetLatestGer(ctx, gerFinalityBlocks)
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("global exit root is updated")
		log.Debugf("rootInContractHash", rootInContractHash)
		log.Debugf("finalGer.GlobalExitRoot", finalGer.GlobalExitRoot)
	}

	return nil, nil
}

func findForcedBatchInL1Logs(ctx context.Context, fromBlock *big.Int, l1 *l1Stuff) (*etrogpolygonzkevm.EtrogpolygonzkevmForceBatch, *types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		Addresses: []common.Address{l1.zkEvmAddr},
	}

	found := false
	for found != true {
		log.Debugf("Looking for forced batch in logs from block %s", fromBlock.String())
		logs, err := l1.ethClient.FilterLogs(ctx, query)
		if err != nil {
			log.Fatal(err)
		}
		for _, vLog := range logs {
			if vLog.Topics[0] == constants.ForcedBatchSignatureHash {
				fb, err := l1.zkEvm.ParseForceBatch(vLog)
				return fb, &vLog, err
			}
		}
		log.Info("Forced batch not found in logs. Waiting 1 second...")
		time.Sleep(1 * time.Second)
	}
	return nil, nil, nil

}
