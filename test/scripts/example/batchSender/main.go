package main

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/hex"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"time"
)

const (
	DefaultL1NetworkURL        = "http://localhost:8545"
	DefaultL1ChainID    uint64 = 1337

	DefaultDeployerAddress     = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	DefaultDeployerPrivateKey  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	DefaultSequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
	DefaultSequencerPrivateKey = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"

	DefaultL1ZkEVMSmartContract         = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
	DefaultL1RollupManagerSmartContract = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
	DefaultL1PolSmartContract           = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
	DefaultL1GERManagerSmartContract    = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
)

const gerFinalityBlocks = uint64(9223372036854775807)

func main() {
	ctx := context.Background()

	opsCfg := operations.GetDefaultOperationsConfig()
	opsCfg.State.MaxCumulativeGasUsed = 80000000000
	var opsman *operations.Manager
	log.Info("Using pre-launched dockers: no reset Database")
	opsman, err := operations.NewManagerNoInitDB(ctx, opsCfg)
	if err != nil {
		log.Fatal(err)
	}

	initialGer, _, err := opsman.State().GetLatestGer(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("BlockNumber: %v", initialGer.BlockNumber)
	log.Infof("GlobalExitRoot: %s", initialGer.GlobalExitRoot)
	log.Infof("MainnetExitRoot: %s", initialGer.MainnetExitRoot)
	log.Infof("RollupExitRoot: %s", initialGer.RollupExitRoot)

	l1Info, err := opsman.State().GetLatestL1InfoRoot(ctx, gerFinalityBlocks)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("BlockNumber: %v", l1Info.GlobalExitRoot.BlockNumber)
	log.Infof("GlobalExitRoot: %s", l1Info.GlobalExitRoot.GlobalExitRoot)
	log.Infof("MainnetExitRoot: %s", l1Info.GlobalExitRoot.MainnetExitRoot)
	log.Infof("RollupExitRoot: %s", l1Info.GlobalExitRoot.RollupExitRoot)

	// Load account with balance on local genesis
	auth, err := operations.GetAuth(DefaultDeployerPrivateKey, operations.DefaultL1ChainID)
	if err != nil {
		log.Fatal(err)
	}
	// Load eth client
	client, err := ethclient.Dial(operations.DefaultL1NetworkURL)
	if err != nil {
		log.Fatal(err)
	}

	g, err := etrogpolygonzkevmglobalexitroot.NewEtrogpolygonzkevmglobalexitroot(common.HexToAddress(DefaultL1GERManagerSmartContract), client)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	lastGlobalExitRoot, err := g.GetLastGlobalExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("lastGlobalExitRoot: %s", common.BytesToHash(lastGlobalExitRoot[:]))

	rollupExitRoot, err := g.LastRollupExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("rollupExitRoot: %s", common.BytesToHash(rollupExitRoot[:]))

	mainnetExitRoot, err := g.LastMainnetExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("mainnetExitRoot: %s", common.BytesToHash(mainnetExitRoot[:]))

	rollupManagerAddr := common.HexToAddress(operations.DefaultL1RollupManagerSmartContract)
	rollupManager, err := etrogpolygonrollupmanager.NewEtrogpolygonrollupmanager(rollupManagerAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	amount, err := rollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Forced batch fee: ", amount)

	zkEvmAddr := common.HexToAddress(operations.DefaultL1ZkEVMSmartContract)
	zkEvm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(zkEvmAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Set forced batch address: ", common.Address{})
	tx, err := zkEvm.SetForceBatchAddress(auth, common.Address{})
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	data, err := hex.DecodeString(rawTxs)
	if err != nil {
		log.Fatal(err)
	}
	tx, err = zkEvm.ForceBatch(auth, data, amount)
	if err != nil {
		log.Fatal(err)
	}
	err = operations.WaitTxToBeMined(ctx, client, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	var sequences []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData
	sequences = append(sequences, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	}, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	})
	tx, err = zkEvm.SequenceBatches(auth, sequences, uint64(time.Now().Unix()), uint64(1), auth.From)
	if err != nil {
		log.Fatal(err)
	}
	err = operations.WaitTxToBeMined(ctx, client, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	lastGlobalExitRoot, err = g.GetLastGlobalExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("lastGlobalExitRoot: %s", common.BytesToHash(lastGlobalExitRoot[:]))

	rollupExitRoot, err = g.LastRollupExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("rollupExitRoot: %s", common.BytesToHash(rollupExitRoot[:]))

	mainnetExitRoot, err = g.LastMainnetExitRoot(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Infof("mainnetExitRoot: %s", common.BytesToHash(mainnetExitRoot[:]))

}
