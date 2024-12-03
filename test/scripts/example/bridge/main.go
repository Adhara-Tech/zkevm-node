package main

import (
  "context"
  "github.com/0xPolygonHermez/zkevm-node/config"

  "math/big"
  "time"

  "github.com/ethereum/go-ethereum/core/types"

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
  toAddressHex = "0x4d5Cf5032B2a844602278b01199ED191A86c93ff"
  forkID6      = 6
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

  time.Sleep(2 * time.Second)

  // Encode transactions
  l2.amount = big.NewInt(0).Add(l2.amount, big.NewInt(10))
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
  lastL2BlockTStamp := unsignedTx.Time().Unix()

  // Send Batch
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
  log.Infof("LastGlobalExitRoot: %v", rootInContractHash)

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

  sequences := []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{{
	Transactions: encodedTxs,
  }, {
	Transactions: encodedTxs,
  }}
  tx, err = l1.zkEvm.SequenceBatches(l1.authSequencer, sequences, uint64(lastL2BlockTStamp), uint64(1), l1.authSequencer.From)
  if err != nil {
	log.Fatal(err)
  }
  log.Debug("TX: ", tx.Hash())
  err = operations.WaitTxToBeMined(ctx, l1.ethClient, tx, operations.DefaultTimeoutTxToBeMined)
  if err != nil {
	log.Fatal(err)
  }

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
  polAmount, _ := big.NewInt(0).SetString("19999999999999999999999", 0)
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
  genesisFileAsStr, err := config.LoadGenesisFileAsString("../../../config/test.genesis.config.json")
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
