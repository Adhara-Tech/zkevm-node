package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

  "github.com/0xPolygonHermez/zkevm-node/etherman"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {

  privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}
	ethman, ethBackend, _, br, err := etherman.NewSimulatedEtherman(etherman.Config{ForkIDChunkSize: 10}, auth)
	if err != nil {
		log.Fatal(err)
	}
	err = ethman.AddOrReplaceAuth(*auth)
	if err != nil {
		log.Fatal(err)
	}

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := ethman.EthClient.BlockByNumber(ctx, nil)
	if err != nil {
  		log.Fatal(err)
  	}

	// Make a bridge tx
	auth.Value = big.NewInt(1000000000000000)
	_, err = br.BridgeAsset(auth, 1, auth.From, auth.Value, common.Address{}, true, []byte{})
	if err != nil {
  		log.Fatal(err)
  	}
	ethBackend.Commit()
	auth.Value = big.NewInt(0)

	amount, err := ethman.EtrogRollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
  		log.Fatal(err)
  	}
	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	data, err := hex.DecodeString(rawTxs)
	if err != nil {
  		log.Fatal(err)
  	}
	_, err = ethman.EtrogZkEVM.ForceBatch(auth, data, amount)
	if err != nil {
  		log.Fatal(err)
  	}
	ethBackend.Commit()

	// Now read the event
	currentBlock, err := ethman.EthClient.BlockByNumber(ctx, nil)
	if err != nil {
  		log.Fatal(err)
  	}
	currentBlockNumber := currentBlock.NumberU64()
	blocks, _, err := ethman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &currentBlockNumber)
	if err != nil {
  		log.Fatal(err)
  	}
	log.Info("Blocks: ", blocks)
	var sequences []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData
	sequences = append(sequences, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	}, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	})
	_, err = ethman.EtrogZkEVM.SequenceBatches(auth, sequences, uint64(time.Now().Unix()), uint64(1), auth.From)
	if err != nil {
  		log.Fatal(err)
  	}

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := ethman.EthClient.BlockByNumber(ctx, nil)
	if err != nil {
  		log.Fatal(err)
  	}
	finalBlockNumber := finalBlock.NumberU64()
	blocks, _, err = ethman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	if err != nil {
  		log.Fatal(err)
  	}
	log.Info("Blocks: %+v", blocks)

}