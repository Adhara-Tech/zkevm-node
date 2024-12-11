package main

import (
  "context"
  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/ethclient"
  "math/big"
)

const (
  DefaultDeployerAddress                     = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  DefaultDeployerPrivateKey                  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  DefaultSequencerAddress                    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
  DefaultSequencerPrivateKey                 = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"
  DefaultL1ZkEVMSmartContract                = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
  DefaultL1RollupManagerSmartContract        = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
  DefaultL1PolSmartContract                  = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
  DefaultL1NetworkURL                        = "http://localhost:8545"
  DefaultL1ChainID                    uint64 = 1337
)

func main() {
  ctx := context.Background()

  log.Infof("connecting to %v: %v", "Local L1", DefaultL1NetworkURL)
  client, err := ethclient.Dial(DefaultL1NetworkURL)
  chkErr(err)
  log.Infof("connected")

  auth := operations.MustGetAuth(DefaultDeployerPrivateKey, DefaultL1ChainID)
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

  nTxs := 1 // send 1 tx by default
  txs := make([]*types.Transaction, 0, nTxs)
  for i := 0; i < nTxs; i++ {
	tx := types.NewTransaction(senderNonce+uint64(i), to, amount, gasLimit, gasPrice, nil)
	txs = append(txs, tx)
  }

  err = operations.ApplyL1Txs(ctx, txs, auth, client)
  chkErr(err)

  log.Infof("%d transactions successfully sent", nTxs)
}

func chkErr(err error) {
  if err != nil {
	log.Fatal(err)
  }
}
