package main

import (
	"context"
	"time"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
	"github.com/ethereum/go-ethereum/crypto"

)

// New GER event Etrog
var updateL1InfoTreeSignatureHash = crypto.Keccak256Hash([]byte("UpdateL1InfoTree(bytes32,bytes32)"))
var updateGlobalExitRootSignatureHash                   = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))

const (
  DefaultBridgeAddress                       = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	DefaultSequencerAddress                    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	DefaultSequencerPrivateKey                 = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	DefaultForcedBatchesAddress                = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
	DefaultForcedBatchesPrivateKey             = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
	DefaultSequencerBalance                    = 400000
	DefaultMaxCumulativeGasUsed                = 800000
	DefaultL1ZkEVMSmartContract                = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
	DefaultL1RollupManagerSmartContract        = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
	DefaultL1PolSmartContract                  = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
	DefaultL1NetworkURL                        = "http://localhost:8545"
	DefaultL1NetworkWebSocketURL               = "ws://localhost:8546"
	DefaultL1ChainID                    uint64 = 1337

	miningTimeout      = 180
)


func main() {

	ctx := context.Background()

	log.Infof("connecting to %v: %v", "L1", DefaultL1NetworkURL)
	l1client, err := ethclient.Dial(DefaultL1NetworkURL)
	chkErr(err)
	log.Infof("connected")

	auth := operations.MustGetAuth(DefaultSequencerPrivateKey, DefaultL1ChainID)
	chkErr(err)

	balance, err := l1client.BalanceAt(ctx, auth.From, nil)
	chkErr(err)
	log.Debugf("ETH Balance for %v: %v", auth.From, balance)

  bridge, err := etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(common.HexToAddress(DefaultBridgeAddress), l1client)
	chkErr(err)
  log.Debugf("%v", bridge)

  // Read currentBlock
	initBlock, err := l1client.BlockByNumber(ctx, nil)
	chkErr(err)
	initBlockNumber := initBlock.NumberU64()

  amount := big.NewInt(1000000000000000)
	auth.Value = amount
	tx, err := bridge.BridgeAsset(auth, 1, auth.From, amount, common.Address{}, true, []byte{})
  chkErr(err)

  err = operations.WaitTxToBeMined(ctx, l1client, tx, miningTimeout*time.Second)
  chkErr(err)

  // Now read the event
	finalBlock, err := l1client.BlockByNumber(ctx, nil)
	chkErr(err)
	finalBlockNumber := finalBlock.NumberU64()

	query := ethereum.FilterQuery{
    FromBlock: new(big.Int).SetUint64(initBlockNumber),
    ToBlock:   new(big.Int).SetUint64(finalBlockNumber),
    Addresses: []common.Address{
      common.HexToAddress(DefaultBridgeAddress),
    },
  }

  logs, err := l1client.FilterLogs(ctx, query)
  chkErr(err)
  log.Debugf("%v", logs)
}

func chkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
