package main

import (
  "context"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "math/big"

  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/ethclient"
)

const (
  DefaultL1NetworkURL        = "http://localhost:8545"
  DefaultL1ChainID    uint64 = 1337
  DefaultL2NetworkURL        = "http://localhost:8123"
  DefaultL2ChainID    uint64 = 1001

  DefaultDeployerAddress     = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  DefaultDeployerPrivateKey  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  DefaultSequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
  DefaultSequencerPrivateKey = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"

  DefaultL1GERManagerSmartContract = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
)

func main() {
  ctx := context.Background()

  log.Infof("connecting to %v: %v", "Local L1", DefaultL1NetworkURL)
  l1client, err := ethclient.Dial(DefaultL1NetworkURL)
  chkErr(err)
  log.Infof("connected")

  g, err := etrogpolygonzkevmglobalexitroot.NewEtrogpolygonzkevmglobalexitroot(common.HexToAddress(DefaultL1GERManagerSmartContract), l1client)
  if err != nil {
	log.Fatal("Error: ", err)
  }

  // Check global exit root
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

  log.Infof("connecting to %v: %v", "Local L2", DefaultL2NetworkURL)
  client, err := ethclient.Dial(DefaultL2NetworkURL)
  chkErr(err)
  log.Infof("connected")

  auth := operations.MustGetAuth(DefaultDeployerPrivateKey, DefaultL2ChainID)
  chkErr(err)

  senderBalance, err := client.BalanceAt(ctx, auth.From, nil)
  chkErr(err)
  log.Debugf("ETH Balance for %v: %v", auth.From, senderBalance)

  amount := big.NewInt(10) //nolint:gomnd
  log.Debugf("Transfer Amount: %v", amount)

  senderNonce, err := client.PendingNonceAt(ctx, auth.From)
  chkErr(err)
  log.Debugf("Sender Nonce: %v", senderNonce)

  to := common.HexToAddress(DefaultSequencerAddress)
  log.Infof("Receiver Addr: %v", to.String())

  gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: auth.From, To: &to, Value: amount})
  chkErr(err)

  gasPrice, err := client.SuggestGasPrice(ctx)
  chkErr(err)

  nTxs := 2 // send 1 tx by default
  txs := make([]*types.Transaction, 0, nTxs)
  for i := 0; i < nTxs; i++ {
	tx := types.NewTransaction(senderNonce+uint64(i), to, amount, gasLimit, gasPrice, nil)
	txs = append(txs, tx)
  }

  _, err = operations.ApplyL2Txs(ctx, txs, auth, client, operations.VerifiedConfirmationLevel)
  chkErr(err)

  log.Infof("%d transactions successfully sent", nTxs)

  // Check global exit root
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

func chkErr(err error) {
  if err != nil {
	log.Fatal(err)
  }
}
