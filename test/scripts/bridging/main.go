package main

import (
  "context"
  "time"
  "math/big"

  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/ethereum/go-ethereum/common"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"

  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
)

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

	DefaultL2NetworkURL                        = "http://localhost:8123"
	PermissionlessL2NetworkURL                 = "http://localhost:8125"
	DefaultL2NetworkWebSocketURL               = "ws://localhost:8133"
	PermissionlessL2NetworkWebSocketURL        = "ws://localhost:8135"
	DefaultL2ChainID                    uint64 = 1001

	DefaultTimeoutTxToBeMined = 1 * time.Minute

	DefaultWaitPeriodSendSequence                          = "15s"
	DefaultLastBatchVirtualizationTimeMaxWaitPeriod        = "10s"
	DefaultMaxTxSizeForL1                           uint64 = 131072

	DeployerAccountAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	DeployerAccountPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	L1AccountAddress = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
	L1AccountPrivateKey = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
  L2AccountAddress = "0xc949254d682d8c9ad5682521675b8f43b102aec4"
  L2AccountPrivateKey = "0xdfd01798f92667dbf91df722434e8fbe96af0211d4d1b82bbbbc8f1def7a814f"
)

type Client struct {
	*ethclient.Client
	Bridge       *etrogpolygonzkevmbridge.Etrogpolygonzkevmbridge
	BridgeSCAddr common.Address
	NodeURL      string
}

// NewClient creates client.
func NewClient(ctx context.Context, nodeURL string, bridgeSCAddr common.Address) (*Client, error) {
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, err
	}
	var br *etrogpolygonzkevmbridge.Etrogpolygonzkevmbridge
	if bridgeSCAddr != (common.Address{}) {
		br, err = etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(bridgeSCAddr, client)
	}
	return &Client{
		Client:       client,
		Bridge:       br,
		BridgeSCAddr: bridgeSCAddr,
		NodeURL:      nodeURL,
	}, err
}

func BalanceAt(ctx context.Context, auth *bind.TransactOpts, c *Client) (*big.Int, error) {
	balance, err := c.Client.BalanceAt(ctx, auth.From, nil)
  if err != nil {
   return nil, err
  }
  return balance, nil
}

func DeployERC20(ctx context.Context, name, symbol string, auth *bind.TransactOpts, c *Client) (common.Address, *ERC20.ERC20, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, instance, err := ERC20.DeployERC20(auth, c.Client, name, symbol)
	if err != nil {
		return common.Address{}, nil, err
	}
	err = operations.WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)

	return addr, instance, err
}

func MintERC20(ctx context.Context, erc20Addr common.Address, amount *big.Int, auth *bind.TransactOpts, c *Client) error {
	erc20sc, err := ERC20.NewERC20(erc20Addr, c.Client)
	if err != nil {
		return err
	}
	tx, err := erc20sc.Mint(auth, amount)
	if err != nil {
		return err
	}
	const txMinedTimeoutLimit = 60 * time.Second
	return operations.WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)
}

func main() {

  ctx := context.Background()

  l1Client, err := NewClient(ctx, DefaultL1NetworkURL, common.HexToAddress(DefaultBridgeAddress))
	if err != nil {
		log.Fatal(err)
	}
  log.Debug(l1Client)
	l2Client, err := NewClient(ctx, DefaultL2NetworkURL, common.HexToAddress(DefaultBridgeAddress))
	if err != nil {
		log.Fatal(err)
	}
  log.Debug(l2Client)

  l1Auth := operations.MustGetAuth(DefaultSequencerPrivateKey, DefaultL1ChainID)

  balance, err := BalanceAt(ctx, l1Auth, l1Client)
  if err != nil {
  	log.Fatal(err)
  }
  log.Debugf("ETH Balance for %v: %v", l1Auth.From, balance)

  l1TokenAddr, _, err := DeployERC20(ctx, "CREATED ON L1", "CL1", l1Auth, l1Client)
  if err != nil {
  	log.Fatal(err)
  }
  err = MintERC20(ctx, l1TokenAddr, big.NewInt(999999999999999999), l1Auth, l1Client)
  if err != nil {
    log.Fatal(err)
  }
}