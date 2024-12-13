package main

import (
  "context"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/ethclient"
  "math/big"
  "time"
)

const (
  DefaultL1NetworkURL        = "http://localhost:8545"
  DefaultL1ChainID    uint64 = 1337

  DefaultDeployerAddress     = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  DefaultDeployerPrivateKey  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  DefaultSequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
  DefaultSequencerPrivateKey = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"
  Account1Address            = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
  Account1PrivateKey         = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"

  DefaultL1ZkEVMSmartContract         = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
  DefaultL1RollupManagerSmartContract = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
  DefaultL1PolSmartContract           = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
  DefaultL1GERManagerSmartContract    = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"

  DefaultTimeoutTxToBeMined = 1 * time.Minute
)

func main() {
  ctx := context.Background()

  // Connect to ethereum node
  ethClient, err := ethclient.Dial(DefaultL1NetworkURL)
  if err != nil {
	log.Fatal(err)
  }
  auth, err := operations.GetAuth(DefaultDeployerPrivateKey, DefaultL1ChainID)
  if err != nil {
	log.Fatal(err)
  }
  authForcedBatch, err := operations.GetAuth(Account1PrivateKey, DefaultL1ChainID)
  if err != nil {
	log.Fatal(err)
  }
  polSmc, err := pol.NewPol(common.HexToAddress(DefaultL1PolSmartContract), ethClient)
  if err != nil {
	log.Fatal(err)
  }
  polAmount, _ := big.NewInt(0).SetString("19999999999999999999999", 0)
  log.Debugf("Charging pol")
  txValue, err := polSmc.Transfer(auth, common.HexToAddress(Account1Address), polAmount)
  if err != nil {
	log.Fatal(err)
  }
  log.Debugf("Waiting for tx %s to be mined (transfer of pol)", txValue.Hash().String())
  err = operations.WaitTxToBeMined(ctx, ethClient, txValue, DefaultTimeoutTxToBeMined)
  if err != nil {
	log.Fatal(err)
  }
  balance, err := polSmc.BalanceOf(&bind.CallOpts{Pending: false}, common.HexToAddress(DefaultDeployerAddress))
  if err != nil {
	log.Fatal(err)
  }
  log.Debugf("Account (sequencer) %s pol balance %s", DefaultDeployerAddress, balance.String())

  balance, err = polSmc.BalanceOf(&bind.CallOpts{Pending: false}, common.HexToAddress(Account1Address))
  if err != nil {
	log.Fatal(err)
  }
  log.Debugf("Account (force_batches) %s pol balance %s", Account1Address, balance.String())
  log.Debugf("Approve to zkEVM SMC to spend %s pol", polAmount.String())
  _, err = polSmc.Approve(authForcedBatch, common.HexToAddress(DefaultL1ZkEVMSmartContract), polAmount)
  if err != nil {
	log.Fatal(err)
  }

  zkEvmAddr := common.HexToAddress(DefaultL1ZkEVMSmartContract)
  zkEvm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(zkEvmAddr, ethClient)
  if err != nil {
	log.Fatal(err)
  }
}
