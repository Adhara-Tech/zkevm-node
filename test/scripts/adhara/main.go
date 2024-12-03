package main

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/config"
	"github.com/0xPolygonHermez/zkevm-node/etherman"
	"github.com/0xPolygonHermez/zkevm-node/etherman/etherscan"
	//"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	//"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// dockersArePreLaunched is a flag that indicates if dockers are pre-launched, used for local development
	// avoiding launch time and reset database time at end (so you can check the database after the test)
	dockersArePreLaunched = true
	gerFinalityBlocks     = uint64(9223372036854775807) // The biggest uint64
)

const (
	invalidParamsErrorCode = -32602
	toAddressHex           = "0x4d5Cf5032B2a844602278b01199ED191A86c93ff"
	forkID6                = 6
	rollUpNumber           = 1
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
	l2 := setupL2Environment(ctx)
	l1 := setupL1Environment(ctx)

	rollupManagerAddr, err := l1.zkEvm.RollupManager(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	globalExitRootManagerAddr, err := l1.zkEvm.GlobalExitRootManager(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	l1Config := etherman.L1Config{
		L1ChainID:                 operations.DefaultL1ChainID,
		ZkEVMAddr:                 l1.zkEvmAddr,
		RollupManagerAddr:         rollupManagerAddr,
		GlobalExitRootManagerAddr: globalExitRootManagerAddr,
	}
	ethConfig := etherman.Config{
		URL:              operations.DefaultL1NetworkURL,
		ForkIDChunkSize:  20000,
		MultiGasProvider: false,
		Etherscan:        etherscan.Config{},
	}
	eth, err := etherman.NewClient(ethConfig, l1Config)
	if err != nil {
		log.Fatal(err)
	}
	auth, err := operations.GetAuth(operations.DefaultSequencerPrivateKey, operations.DefaultL1ChainID)
	if err != nil {
		log.Fatal(err)
	}
	err = eth.AddOrReplaceAuth(*auth)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Using address: ", auth.From)

	_, l2BlockNumbers, l1Blocks := sendL2Transfers(ctx, eth, l2)
	checkL2Batches(ctx, l2BlockNumbers, l1Blocks, eth, l1, l2)

	checkL1State(ctx, l1)
	checkL2State(ctx, l2)

	/*
		encodedTxs := generateSignedAndEncodedTxs(ctx, l2)
		_, err := sendSequencedBatch(ctx, encodedTxs, l2.opsman, l1, rollUpNumber, 1)
		if err != nil {
			log.Fatal(err)
		}

		encodedTxs = generateSignedAndEncodedTxs(ctx, l2)
		_, err = sendSequencedBatch(ctx, encodedTxs, l2.opsman, l1, rollUpNumber+1, 2)
		if err != nil {
			log.Fatal(err)
		}

		encodedTxs = generateSignedAndEncodedTxs(ctx, l2)
		_, err = sendSequencedBatch(ctx, encodedTxs, l2.opsman, l1, rollUpNumber+2, 3)
		if err != nil {
			log.Fatal(err)
		}
	*/
}

func sendL2Transfers(ctx context.Context, eth *etherman.Client, l2 *l2Stuff) ([]*types.Transaction, []*big.Int, []etherman.Block) {
	initBlock, err := eth.EthClient.BlockByNumber(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	st := l2.opsman.State()
	numTxs := 2
	addr := common.HexToAddress(toAddressHex)
	lastL2Block, err := st.GetLastL2Block(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	lastL2BlockNumber := lastL2Block.Block.Number().Uint64()
	log.Infof("Last L2 block: %d", lastL2BlockNumber)
	toL2Balance, err := st.GetBalance(ctx, addr, lastL2Block.Root())
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Receiving account balance on L2 at block %d: %d", lastL2BlockNumber, toL2Balance)

	log.Infof("Sending %d txs and waiting until added into the trusted sequencer pool", numTxs)
	toAddress := common.HexToAddress(toAddressHex)
	txs := make([]*types.Transaction, 0, numTxs)
	for i := 0; i < numTxs; i++ {
		amount := big.NewInt(l2.amount.Int64() + int64(i))
		tx := types.NewTx(&types.LegacyTx{Nonce: l2.nonce, GasPrice: l2.gasPrice, Gas: l2.gasLimit, To: &toAddress, Value: amount})
		l2.nonce++
		txs = append(txs, tx)
	}
	var blockNumbers []*big.Int
	blockNumbers, err = operations.ApplyL2Txs(ctx, txs, l2.authSequencer, l2.client, operations.VerifiedConfirmationLevel)
	if err != nil {
		log.Fatal(err)
	}
	checkKeys := make(map[string]bool)
	var blockNumberArray []*big.Int
	for _, val := range blockNumbers {
		key := val.String()
		_, have := checkKeys[key]
		if !have {
			checkKeys[key] = true
			blockNumberArray = append(blockNumberArray, val)
		}
	}
	lastL2Block, err = st.GetLastL2Block(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	lastL2BlockNumber = lastL2Block.Block.Number().Uint64()
	log.Infof("Last L2 block: %d", lastL2BlockNumber)
	toL2Balance, err = st.GetBalance(ctx, addr, lastL2Block.Root())
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Receiving account balance on L2 at block %d: %d", lastL2BlockNumber, toL2Balance)

	finalBlock, err := eth.EthClient.BlockByNumber(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	finalBlockNumber := finalBlock.NumberU64()
	blocks, _, err := eth.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	log.Infof("Blocks: %+v", blocks)

	return txs, blockNumberArray, blocks
}

func checkL2Batches(ctx context.Context, blockNumbers []*big.Int, blocks []etherman.Block, eth *etherman.Client, l1 *l1Stuff, l2 *l2Stuff) {
	st := l2.opsman.State()
	lastL2Block, err := st.GetLastL2Block(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	lastL2BlockNumber := lastL2Block.Block.Number().Uint64()
	log.Infof("Last L2 block: %d", lastL2BlockNumber)

	for _, blockNumber := range blockNumbers {
		//l2Block, err := st.GetBlockByNumber(ctx, blockNumber.Uint64(), nil)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//log.Infof("Hash for block %d: %d", blockNumber.Uint64(), l2Block.BlockHash)
		batch, err := st.GetBatchByL2BlockNumber(ctx, blockNumber.Uint64(), nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Batch for detected at block %d on L2", blockNumber.Uint64())
		log.Infof("                  : Batch number    : %d", batch.BatchNumber)
		log.Infof("                  : State root      : %s", batch.StateRoot)
		log.Infof("                  : Local exit root : %s", batch.LocalExitRoot)
		log.Infof("                  : Global exit root: %s", batch.GlobalExitRoot)
		for _, block := range blocks {
			for _, verified := range block.VerifiedBatches {
				if verified.BatchNumber == batch.BatchNumber {
					log.Infof("Batch was verified at block %d on L1", block.BlockNumber)
				}
			}
		}
	}
}

func checkL2State(ctx context.Context, l2 *l2Stuff) {

	st := l2.opsman.State()
	lastL2GlobalExitRoot, _, err := st.GetLatestGer(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Last L2 ger: %s", lastL2GlobalExitRoot.GlobalExitRoot.String())
	log.Infof("Last L2 mer: %s", lastL2GlobalExitRoot.MainnetExitRoot.String())
	log.Infof("Last L2 rer: %s", lastL2GlobalExitRoot.RollupExitRoot.String())
	log.Infof("Last L2 bnr: %d", lastL2GlobalExitRoot.BlockNumber)
	l1Info, err := st.GetLatestL1InfoRoot(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Last L1 info tree: ger: %s", l1Info.GlobalExitRoot.GlobalExitRoot.String())
	log.Infof("Last L1 info tree: mer: %s", l1Info.GlobalExitRoot.MainnetExitRoot.String())
	log.Infof("Last L1 info tree: rer: %s", l1Info.GlobalExitRoot.RollupExitRoot.String())
	log.Infof("Last L1 info tree: bnr: %d", l1Info.GlobalExitRoot.BlockNumber)
	log.Info("\n")
}

func checkL1State(ctx context.Context, l1 *l1Stuff) {
	//initialBlock, err := l1.ethClient.BlockByNumber(ctx, big.NewInt(0))
	//if err != nil {
	//	log.Fatal(err)
	//}
	// log.Infof("First L1 block: %d ", initialBlock.Number().Uint64())

	lastL1BlockNumber, err := l1.ethClient.BlockNumber(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("L1: Last block number: %d", lastL1BlockNumber)

	gerManagerAddress, err := l1.zkEvm.GlobalExitRootManager(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("L1 GERM: Contract address: %s", gerManagerAddress.Hex())
	//gerManager, err := etrogpolygonzkevmglobalexitroot.NewEtrogpolygonzkevmglobalexitroot(gerManagerAddress, l1.ethClient)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//bridgeAddress, err := gerManager.BridgeAddress(&bind.CallOpts{Pending: false})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Infof("L1 GERM: Bridge address: %s", bridgeAddress.Hex())
	//
	//lastL1GlobalExitRoot, err := gerManager.GetLastGlobalExitRoot(&bind.CallOpts{Pending: false})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//lastL1GlobalExitRootHash := common.BytesToHash(lastL1GlobalExitRoot[:])
	//log.Infof("L1 GERM: Latest global exit root: %s", lastL1GlobalExitRootHash)
	//lastL1RollupExitRoot, err := gerManager.LastRollupExitRoot(&bind.CallOpts{Pending: false})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//lastL1RollupExitRootHash := common.BytesToHash(lastL1RollupExitRoot[:])
	//log.Infof("L1 GERM: Latest rollup exit root: %s", lastL1RollupExitRootHash)
	////gerManager.VerifyMerkleProof()
	//log.Info("\n")
}

func generateEncodedTxs(ctx context.Context, l2 *l2Stuff) []byte {
	toAddress := common.HexToAddress(toAddressHex)
	var txs []types.Transaction
	var effectivePercentages []uint8
	for i := 0; i < 4; i++ {
		amount := big.NewInt(l2.amount.Int64() + int64(i))
		unsignedTx := types.NewTx(&types.LegacyTx{Nonce: l2.nonce, GasPrice: l2.gasPrice, Gas: l2.gasLimit, To: &toAddress, Value: amount})
		l2.nonce++
		signedTx, err := l2.authSequencer.Signer(l2.authSequencer.From, unsignedTx)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Batch: tx %d -> %s", i, signedTx.Hash().String())
		txs = append(txs, *signedTx)
		effectivePercentages = append(effectivePercentages, state.MaxEffectivePercentage)
	}
	encodedTxs, err := state.EncodeTransactions(txs, effectivePercentages, forkID6)
	if err != nil {
		log.Fatal(err)
	}
	return encodedTxs
}

func setupL2Environment(ctx context.Context) *l2Stuff {
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

func setupL1Environment(ctx context.Context) *l1Stuff {
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

/*
func sendSequencedBatch(ctx context.Context, txs []byte, opsman *operations.Manager, l1 *l1Stuff, rollUp uint32, batchNum uint64) (*state.Batch, error) {
	st := opsman.State()

	initialBlock, err := l1.ethClient.BlockByNumber(ctx, nil)
	log.Info("Block start ", initialBlock.Number())

	initialGer, _, err := st.GetLatestGer(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Using address: ", l1.authForcedBatch.From)

	rollupManagerAddr := common.HexToAddress(operations.DefaultL1RollupManagerSmartContract)
	rollupManager, err := etrogpolygonrollupmanager.NewEtrogpolygonrollupmanager(rollupManagerAddr, l1.ethClient)
	if err != nil {
		log.Fatal(err)
	}
	//lastVerifiedBatchNumber, err := rollupManager.GetLastVerifiedBatch(&bind.CallOpts{Pending: false}, rollUp-1)
	//log.Debugf("Last verified batch: %d", lastVerifiedBatchNumber)

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
	if rootInContractHash != initialGer.GlobalExitRoot {
		log.Info("Checking if global exit root is updated...")
		finalGer, _, err := st.GetLatestGer(ctx, gerFinalityBlocks)
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("Global exit root is updated")
		log.Debugf("Last GlobalExitRoot   : %v", rootInContractHash)
		log.Debugf("Current GlobalExitRoot: %v", finalGer.GlobalExitRoot)
	}

	var sequences []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData
	sequences = append(sequences, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: txs,
	})

	//log.Infof("Activating forced batches...")
	//_, err = l1.zkEvm.SetForceBatchAddress(l1.authSequencer, common.Address{})
	//if err != nil {
	//	log.Fatal(err)
	//}

	tx, err := l1.zkEvm.SequenceBatches(l1.authSequencer, sequences, uint64(time.Now().Unix()), batchNum, l1.authSequencer.From)
	if err != nil {
		log.Debugf("Invoking 'SequenceBatches' on L1 failed with code [%d] and data [%s]: %s", err.(rpc.Error).ErrorCode(), err.(rpc.DataError).ErrorData(), err.(rpc.Error).Error())
		log.Fatal(err)
	}
	log.Debug("Sequence batches tx: ", tx.Hash())
	err = operations.WaitTxToBeMined(ctx, l1.ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Duration(time.Second * 5))

	lastVerifiedBatchNumber, err := rollupManager.GetLastVerifiedBatch(&bind.CallOpts{Pending: false}, rollUp)
	log.Debugf("Last verified batch: %d", lastVerifiedBatchNumber)

	stateRoot, err := rollupManager.GetRollupBatchNumToStateRoot(&bind.CallOpts{Pending: false}, rollUp, lastVerifiedBatchNumber)
	log.Debugf("State root: %s ", common.BytesToHash(stateRoot[:]).String())

	return nil, nil
}

func findSequenceBatchesInL1Logs(ctx context.Context, fromBlock *big.Int, l1 *l1Stuff) (*etrogpolygonzkevm.EtrogpolygonzkevmSequenceBatches, *types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		Addresses: []common.Address{l1.zkEvmAddr},
	}

	found := false
	for found != true {
		log.Debugf("Looking for sequence batches in logs from block %s", fromBlock.String())
		logs, err := l1.ethClient.FilterLogs(ctx, query)
		if err != nil {
			log.Fatal(err)
		}
		for _, vLog := range logs {
			if vLog.Topics[0] == constants.SequenceBatchesSignatureHash {
				fb, err := l1.zkEvm.ParseSequenceBatches(vLog)
				return fb, &vLog, err
			}
		}
		log.Info("Sequence batches not found in logs. Waiting 1 second...")
		time.Sleep(1 * time.Second)
	}
	return nil, nil, nil

}
*/
/*
  {
    "id": 0,
    "description": "2 accounts and 1 valid transaction.",
    "arity": 4,
    "chainIdSequencer": 1001,
    "defaultChainId": 1000,
    "sequencerAddress": "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D",
    "sequencerPvtKey": "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e",
    "genesis": [
      {
        "address": "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D",
        "nonce": "0",
        "balance": "100000000000000000000",
        "pvtKey": "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"
      },
      {
        "address": "0x4d5Cf5032B2a844602278b01199ED191A86c93ff",
        "nonce": "0",
        "balance": "200000000000000000000",
        "pvtKey": "0x4d27a600dce8c29b7bd080e29a26972377dbb04d7a27d919adbb602bf13cfd23"
      }
    ],
    "expectedOldRoot": "0xbdde84a5932a2f0a1a4c6c51f3b64ea265d4f1461749298cfdd09b31122ce0d6",
    "txs": [
      {
        "id": 0,
        "from": "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D",
        "to": "0x4d5Cf5032B2a844602278b01199ED191A86c93ff",
        "nonce": 0,
        "value": "100000000000000000",
        "gasLimit": 100000,
        "gasPrice": "1000000000",
        "chainId": 1001,
        "rawTx": "0xf86d80843b9aca00830186a0944d5cf5032b2a844602278b01199ed191a86c93ff88016345785d8a0000808207f5a01186622d03b6b8da7cf111d1ccba5bb185c56deae6a322cebc6dda0556f3cb979f910c26408b64b51c5da36ba2f38ef55ba1cee719d5a6c01225968799907432",
        "customRawTx": "0xee80843b9aca00830186a0944d5cf5032b2a844602278b01199ed191a86c93ff88016345785d8a0000808203e980801186622d03b6b8da7cf111d1ccba5bb185c56deae6a322cebc6dda0556f3cb9700910c26408b64b51c5da36ba2f38ef55ba1cee719d5a6c012259687999074321b",
        "reason": ""
      }
    ],
    "expectedNewRoot": "0xc0465433d9caa28474892b15dd2266db120036294da4cf8e4cde9e5e68898935",
    "expectedNewLeafs": {
      "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D": {
        "balance": "99900000000000000000",
        "nonce": "1"
      },
      "0x4d5Cf5032B2a844602278b01199ED191A86c93ff": {
        "balance": "200100000000000000000",
        "nonce": "0"
      }
    },
    "batchL2Data": "0xee80843b9aca00830186a0944d5cf5032b2a844602278b01199ed191a86c93ff88016345785d8a0000808203e980801186622d03b6b8da7cf111d1ccba5bb185c56deae6a322cebc6dda0556f3cb9700910c26408b64b51c5da36ba2f38ef55ba1cee719d5a6c012259687999074321b",
    "globalExitRoot": "0x090bcaf734c4f06c93954a827b45a6e8c67b8e0fd1e0a35a1c5982d6961828f9",
    "newLocalExitRoot": "0x17c04c3760510b48c6012742c540a81aba4bca2f78b9d14bfd2f123e2e53ea3e",
    "inputHash": "0xa2d6f70384a5211fc6c8cf1698132ee9e8cf83943004448c0065dc727dee5c3a",
    "batchHashData": "0x3d53e7e5be04b00f66af647512af6d17e4e767a5e41fa1293010b885c9fe06db",
    "oldLocalExitRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "timestamp": 1944498031,
    "receipts": [
      {
        "txId": 0,
        "receipt": {
          "transactionHash": "0x0cc3dd49b941271b19df83ba6733bed4023fb82c28d40e6ef863ca589d4a933a",
          "transactionIndex": 0,
          "blockNumber": 0,
          "from": "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D",
          "to": "0x4d5Cf5032B2a844602278b01199ED191A86c93ff",
          "cumulativeGasUsed": 21000,
          "gasUsedForTx": 21000,
          "contractAddress": null,
          "logs": 0,
          "logsBloom": 0,
          "status": 1,
          "blockHash": "0xd428bcb4a86d605119259aa806372e61a98b210f940d0007a510954ebf01d698"
        }
      }
    ],
    "blockInfo": {
      "blockNumber": 0,
      "gasUsedForTx": 21000,
      "blockGasLimit": 30000000,
      "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "txHashRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "receiptRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "timestamp": 1944498031
    }
  },
*/
