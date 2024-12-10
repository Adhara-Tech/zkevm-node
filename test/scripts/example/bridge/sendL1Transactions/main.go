package main

import (
  "context"
  "fmt"
  "github.com/0xPolygonHermez/zkevm-node/hex"
  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
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

  balance, err := client.BalanceAt(ctx, auth.From, nil)
  chkErr(err)
  log.Debugf("ETH Balance for %v: %v", auth.From, balance)

  // Valid ETH Transfer
  balance, err = client.BalanceAt(ctx, auth.From, nil)
  log.Debugf("ETH Balance for %v: %v", auth.From, balance)
  chkErr(err)
  transferAmount := big.NewInt(1)
  log.Debugf("Transfer Amount: %v", transferAmount)

  nonce, err := client.NonceAt(ctx, auth.From, nil)
  chkErr(err)
  // var lastTxHash common.Hash
  for i := 0; i < 1000; i++ {
	nonce := nonce + uint64(i)
	log.Debugf("Sending TX to transfer ETH")
	to := common.HexToAddress(DefaultSequencerAddress)
	tx := ethTransfer(ctx, client, auth, to, transferAmount, &nonce)
	fmt.Println("tx sent: ", tx.Hash().String())
  }
}

func ethTransfer(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, to common.Address, amount *big.Int, nonce *uint64) *types.Transaction {
  if nonce == nil {
	log.Infof("reading nonce for account: %v", auth.From.Hex())
	var err error
	n, err := client.NonceAt(ctx, auth.From, nil)
	log.Infof("nonce: %v", n)
	chkErr(err)
	nonce = &n
  }

  gasPrice, err := client.SuggestGasPrice(context.Background())
  chkErr(err)

  gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{To: &to})
  chkErr(err)

  tx := types.NewTransaction(*nonce, to, amount, gasLimit, gasPrice, nil)

  signedTx, err := auth.Signer(auth.From, tx)
  chkErr(err)

  log.Infof("sending transfer tx")
  err = client.SendTransaction(ctx, signedTx)
  chkErr(err)
  log.Infof("tx sent: %v", signedTx.Hash().Hex())

  rlp, err := signedTx.MarshalBinary()
  chkErr(err)

  log.Infof("tx rlp: %v", hex.EncodeToHex(rlp))

  return signedTx
}

func chkErr(err error) {
  if err != nil {
	log.Fatal(err)
  }
}
