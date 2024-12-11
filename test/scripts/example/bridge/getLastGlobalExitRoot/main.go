package main

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
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
	DefaultL1BridgeSmartContract     = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
)

func main() {
	ctx := context.Background()

	client, err := ethclient.Dial(DefaultL1NetworkURL)
	if err != nil {
		log.Fatal("error connecting to the node. Error: ", err)
	}

	auth := operations.MustGetAuth(DefaultDeployerPrivateKey, DefaultL1ChainID)
	if err != nil {
		log.Fatal("Error: ", err)
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

	bridgeAddr := common.HexToAddress(DefaultL1BridgeSmartContract)
	bridge, err := etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(bridgeAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	// Make a bridge tx
	auth.Value = big.NewInt(1000000000000000)
	tx, err := bridge.BridgeAsset(auth, 1, auth.From, auth.Value, common.Address{}, true, []byte{})
	if err != nil {
		log.Fatal(err)
	}
	err = operations.WaitTxToBeMined(ctx, client, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}
	auth.Value = big.NewInt(0)

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
