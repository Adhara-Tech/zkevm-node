package main

import (
	"context"
	"time"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"

	"github.com/ethereum/go-ethereum"

  //"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonrollupmanager"
  //"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

)

const (
  l1NetworkURL                        = "http://localhost:8545"
  l1ChainID                    uint64 = 1337

	// PolTokenAddress token address
	PolTokenAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
	l1BridgeAddr    = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	l2BridgeAddr    = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"

	l1AccHexAddress = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
  l1AccHexPrivateKey = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" //0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC

  l2AccHexAddress = "0xc949254d682d8c9ad5682521675b8f43b102aec4"
  l2AccHexPrivateKey = "0xdfd01798f92667dbf91df722434e8fbe96af0211d4d1b82bbbbc8f1def7a814f" //0xc949254d682d8c9ad5682521675b8f43b102aec4

  deployerAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  deployerHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" //0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

)

var scAddresses = []common.Address{
  common.HexToAddress(l1BridgeAddr),
}

var(
  updateL1InfoTreeSignatureHash = crypto.Keccak256Hash([]byte("UpdateL1InfoTree(bytes32,bytes32)"))
  depositEventSignatureHash = crypto.Keccak256Hash([]byte("BridgeEvent(uint8,uint32,address,uint32,address,uint256,bytes,uint32)"))
)

// Block struct
type Block struct {
	Deposits        []Deposit
}

// Deposit struct
type Deposit struct {
	Id                 uint64
	LeafType           uint8
	OriginalNetwork    uint
	OriginalAddress    common.Address
	Amount             *big.Int
	DestinationNetwork uint
	DestinationAddress common.Address
	DepositCount       uint
	BlockID            uint64
	BlockNumber        uint64
	NetworkID          uint
	TxHash             common.Hash
	Metadata           []byte
	// it is only used for the bridge service
	ReadyForClaim bool
}

var block Block

func main() {

	ctx := context.Background()

	log.Infof("connecting to %v: %v", "L1", l1NetworkURL)
	client, err := ethclient.Dial(l1NetworkURL)
	chkErr(err)
	log.Infof("connected")

	auth := operations.MustGetAuth(l1AccHexPrivateKey, l1ChainID)
	chkErr(err)
	log.Debugf("auth.from: %v", auth.From)

  balance, err := client.BalanceAt(ctx, common.HexToAddress(l1AccHexAddress), nil)
  chkErr(err)
  log.Debugf("ETH Balance for l1AccHexAddress %v: %v", l1AccHexAddress, balance)

  tokenAddr, tx, erc20sc, err := ERC20.DeployERC20(auth, client, "A COIN", "ACO")
  chkErr(err)
	err = operations.WaitTxToBeMined(ctx, client, tx, 60*time.Second)
  chkErr(err)
	log.Info("Token Addr: ", tokenAddr.Hex())
	amountTokens := new(big.Int).SetUint64(1000000000000000000)
	tx, err = erc20sc.Approve(auth, common.HexToAddress(l1BridgeAddr), amountTokens)
  chkErr(err)
  err = operations.WaitTxToBeMined(ctx, client, tx, 60*time.Second)
  chkErr(err)
	tx, err = erc20sc.Mint(auth, amountTokens)
	chkErr(err)
  err = operations.WaitTxToBeMined(ctx, client, tx, 60*time.Second)
  chkErr(err)

  balance, err = erc20sc.BalanceOf(&bind.CallOpts{Pending: false}, auth.From)
  chkErr(err)
  log.Debugf("ETH Balance for %v: %v", auth.From, balance)

  // Read currentBlock
  initBlock, err := client.BlockByNumber(ctx, nil)
  chkErr(err)
  initBlockNumber := initBlock.NumberU64()

  // Make a bridge tx
  bridge, err := etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(common.HexToAddress(l1BridgeAddr), client)
  amount := big.NewInt(90000000000000000)
  tx, err = bridge.BridgeAsset(auth, 1, common.HexToAddress(l1AccHexAddress), amount, tokenAddr, true, []byte{})
  chkErr(err)

  err = operations.WaitTxToBeMined(ctx, client, tx, 60*time.Second)
  chkErr(err)

  // Now read the event
  finalBlock, err := client.BlockByNumber(ctx, nil)
  chkErr(err)
  finalBlockNumber := finalBlock.NumberU64()

  query := ethereum.FilterQuery{
    FromBlock: new(big.Int).SetUint64(initBlockNumber),
    ToBlock:   new(big.Int).SetUint64(finalBlockNumber),
    Addresses: scAddresses,
  }

  logs, err := client.FilterLogs(ctx, query)
  chkErr(err)

  for _, vLog := range logs {
    switch vLog.Topics[0] {
     case updateL1InfoTreeSignatureHash:
	     log.Infof("UpdateL1InfoTree event detected: %v", vLog.Topics[0])
     case depositEventSignatureHash:
	     log.Infof("Deposit event detected: %v", vLog.Topics[0])
	     d, err := bridge.ParseBridgeEvent(vLog)
	     chkErr(err)
	     var deposit Deposit
       deposit.Amount = d.Amount
       deposit.BlockNumber = vLog.BlockNumber
       deposit.OriginalNetwork = uint(d.OriginNetwork)
       deposit.DestinationAddress = d.DestinationAddress
       deposit.DestinationNetwork = uint(d.DestinationNetwork)
       deposit.OriginalAddress = d.OriginAddress
       deposit.DepositCount = uint(d.DepositCount)
       deposit.TxHash = vLog.TxHash
       deposit.Metadata = d.Metadata
       deposit.LeafType = d.LeafType
       block.Deposits = append(block.Deposits, deposit)
  	default:
	     log.Infof("Event not registered: %+v", vLog.Topics[0])
    }
	}

  GlobalExitRootManAddr, err := bridge.GlobalExitRootManager(&bind.CallOpts{Pending: false})
  chkErr(err)
  log.Debugf("GlobalExitRootManAddr: %v", GlobalExitRootManAddr)

  globalManager, err := etrogpolygonzkevmglobalexitroot.NewEtrogpolygonzkevmglobalexitroot(GlobalExitRootManAddr, client)
  chkErr(err)
  log.Debugf("globalManager: %v", globalManager)
  gMainnet, err := globalManager.LastMainnetExitRoot(&bind.CallOpts{Pending: false})
  chkErr(err)
  log.Debugf("gMainnet: %v", gMainnet)
  gRollup, err := globalManager.LastRollupExitRoot(&bind.CallOpts{Pending: false})
  chkErr(err)
  log.Debugf("gRollup: %v", gRollup)

}

func chkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}