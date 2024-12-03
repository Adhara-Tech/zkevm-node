package main

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	//"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

const DefaultL1BridgeSmartContract = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"

func main() {
	ctx := context.Background()
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

	polAddr := common.HexToAddress(operations.DefaultL1PolSmartContract)
	polSmc, err := pol.NewPol(polAddr, ethClient)
	if err != nil {
		log.Fatal(err)
	}
	polAmount, _ := big.NewInt(0).SetString("90000000000000000", 0)
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
	log.Debugf("Approve to Bridge SMC to spend %s pol", polAmount.String())
	tx, err := polSmc.Approve(authForcedBatch, common.HexToAddress(DefaultL1BridgeSmartContract), polAmount)
	if err != nil {
		log.Fatal(err)
	}
	err = operations.WaitTxToBeMined(ctx, ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	bridgeAddr := common.HexToAddress(DefaultL1BridgeSmartContract)
	bridge, err := etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(bridgeAddr, ethClient)
	if err != nil {
		log.Fatal(err)
	}

	destAddr := common.HexToAddress(operations.DefaultForcedBatchesAddress)
	tx, err = bridge.BridgeAsset(authForcedBatch, 1, destAddr, polAmount, polAddr, true, []byte{})
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("Bridge Asset Sent")
	//zkEvmAddr := common.HexToAddress(operations.DefaultL1ZkEVMSmartContract)
	//zkEvm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(zkEvmAddr, ethClient)
	//if err != nil {
	//	log.Fatal(err)
	//}

}
