package main

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/0xPolygonHermez/zkevm-node/test/constants"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
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
const toAddressHex = "0x4d5Cf5032B2a844602278b01199ED191A86c93ff"

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
	log.Debugf("Account (forcedBatchesAddress) %s pol balance %s", operations.DefaultForcedBatchesAddress, balance.String())

	brAmount, _ := big.NewInt(0).SetString("1000000000000000", 0)
	log.Debugf("Approve to Bridge SMC to spend %s pol", brAmount.String())
	tx, err := polSmc.Approve(authForcedBatch, common.HexToAddress(DefaultL1BridgeSmartContract), brAmount)
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
	tx, err = bridge.BridgeAsset(authForcedBatch, 1, destAddr, brAmount, polAddr, true, []byte{})
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("Account (forcedBatchesAddress) %s bridge amount %s", operations.DefaultForcedBatchesAddress, brAmount)

	zkEvmAddr := common.HexToAddress(operations.DefaultL1ZkEVMSmartContract)
	zkEvm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(zkEvmAddr, ethClient)
	if err != nil {
		log.Fatal(err)
	}

	// L2 network
	// Load account with balance on local genesis
	auth, err := operations.GetAuth(operations.DefaultSequencerPrivateKey, operations.DefaultL2ChainID)
	if err != nil {
		log.Fatal(err)
	}
	// Load eth client
	client, err := ethclient.Dial(operations.DefaultL2NetworkURL)
	if err != nil {
		log.Fatal(err)
	}
	// Send txs
	senderNonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress(toAddressHex)
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: auth.From, To: &toAddress, Value: big.NewInt(10000)})
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	tx1 := types.NewTransaction(senderNonce, toAddress, big.NewInt(10), gasLimit, gasPrice, []byte{})
	batchL2Data, err := state.EncodeTransactions([]types.Transaction{*tx1}, constants.EffectivePercentage, 6)
	if err != nil {
		log.Fatal(err)
	}
	var batches []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData
	var ger common.Hash
	batch := etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions:         batchL2Data,
		ForcedGlobalExitRoot: ger,
	}
	batches = append(batches, batch)
	lastL2BlockTStamp := tx1.Time().Unix()

	log.Infof("Activating forced batches...")
	tx, err = zkEvm.SetForceBatchAddress(authForcedBatch, destAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Forced batch is disallowed. Activated. Waiting for tx %s to be mined", tx.Hash())
	err = operations.WaitTxToBeMined(ctx, ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

	tx, err = zkEvm.SequenceBatches(authForcedBatch, batches, uint64(lastL2BlockTStamp), uint64(1), destAddr)
	if err != nil {
		log.Fatal(err)
	}
	err = operations.WaitTxToBeMined(ctx, ethClient, tx, operations.DefaultTimeoutTxToBeMined)
	if err != nil {
		log.Fatal(err)
	}

}
