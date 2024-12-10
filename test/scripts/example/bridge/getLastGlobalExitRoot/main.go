package main

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	DefaultL1GERManagerSmartContract        = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
	DefaultL1NetworkURL                     = "http://localhost:8545"
	DefaultL1ChainID                 uint64 = 1337

	DefaultL2NetworkURL        = "http://localhost:8123"
	DefaultL2ChainID    uint64 = 1001

	DefaultSequencerAccount    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
	DefaultSequencerPrivateKey = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"
)

func main() {
	ctx := context.Background()

	client, err := ethclient.Dial(DefaultL2NetworkURL)
	if err != nil {
		log.Fatal("error connecting to the node. Error: ", err)
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

}
