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

  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonrollupmanager"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

)

const (
  DefaultGlobalExitRootManager               = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
	DefaultSequencerAddress                    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	DefaultSequencerPrivateKey                 = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	DefaultForcedBatchesAddress                = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
	DefaultForcedBatchesPrivateKey             = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
	DefaultSequencerBalance                    = 400000
	DefaultMaxCumulativeGasUsed                = 800000
	DefaultL1ZkEVMSmartContract                = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
	DefaultL1RollupManagerSmartContract        = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
	DefaultL1BridgeAddressSmartContract        = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	DefaultL1PolSmartContract                  = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
	DefaultL1NetworkURL                        = "http://localhost:8545"
	DefaultL1NetworkWebSocketURL               = "ws://localhost:8546"
	DefaultL1ChainID                    uint64 = 1337

	miningTimeout      = 180
)

var(
  //updateL1InfoTreeSignatureHash = crypto.Keccak256Hash([]byte("UpdateL1InfoTree(bytes32,bytes32)"))
  //updateGlobalExitRootSignatureHash = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
  depositEventSignatureHash = crypto.Keccak256Hash([]byte("BridgeEvent(uint8,uint32,address,uint32,address,uint256,bytes,uint32)"))
  //forceBatchSignatureHash = crypto.Keccak256Hash([]byte("ForceBatch(uint64,bytes32,address,bytes)"))
  //onSequenceBatchesSignatureHash = crypto.Keccak256Hash([]byte("OnSequenceBatches(uint32,uint64)"))
  //sequenceBatchesSignatureHash = crypto.Keccak256Hash([]byte("SequenceBatches(uint64,bytes32)"))
)

var (
	// Events EtrogRollupManager
	setBatchFeeSignatureHash                       = crypto.Keccak256Hash([]byte("SetBatchFee(uint256)"))
	setTrustedAggregatorSignatureHash              = crypto.Keccak256Hash([]byte("SetTrustedAggregator(address)"))       // Used in oldZkEvm as well
	setVerifyBatchTimeTargetSignatureHash          = crypto.Keccak256Hash([]byte("SetVerifyBatchTimeTarget(uint64)"))    // Used in oldZkEvm as well
	setMultiplierBatchFeeSignatureHash             = crypto.Keccak256Hash([]byte("SetMultiplierBatchFee(uint16)"))       // Used in oldZkEvm as well
	setPendingStateTimeoutSignatureHash            = crypto.Keccak256Hash([]byte("SetPendingStateTimeout(uint64)"))      // Used in oldZkEvm as well
	setTrustedAggregatorTimeoutSignatureHash       = crypto.Keccak256Hash([]byte("SetTrustedAggregatorTimeout(uint64)")) // Used in oldZkEvm as well
	overridePendingStateSignatureHash              = crypto.Keccak256Hash([]byte("OverridePendingState(uint32,uint64,bytes32,bytes32,address)"))
	proveNonDeterministicPendingStateSignatureHash = crypto.Keccak256Hash([]byte("ProveNonDeterministicPendingState(bytes32,bytes32)")) // Used in oldZkEvm as well
	consolidatePendingStateSignatureHash           = crypto.Keccak256Hash([]byte("ConsolidatePendingState(uint32,uint64,bytes32,bytes32,uint64)"))
	verifyBatchesTrustedAggregatorSignatureHash    = crypto.Keccak256Hash([]byte("VerifyBatchesTrustedAggregator(uint32,uint64,bytes32,bytes32,address)"))
	rollupManagerVerifyBatchesSignatureHash        = crypto.Keccak256Hash([]byte("VerifyBatches(uint32,uint64,bytes32,bytes32,address)"))
	onSequenceBatchesSignatureHash                 = crypto.Keccak256Hash([]byte("OnSequenceBatches(uint32,uint64)"))
	updateRollupSignatureHash                      = crypto.Keccak256Hash([]byte("UpdateRollup(uint32,uint32,uint64)"))
	addExistingRollupSignatureHash                 = crypto.Keccak256Hash([]byte("AddExistingRollup(uint32,uint64,address,uint64,uint8,uint64)"))
	createNewRollupSignatureHash                   = crypto.Keccak256Hash([]byte("CreateNewRollup(uint32,uint32,address,uint64,address)"))
	obsoleteRollupTypeSignatureHash                = crypto.Keccak256Hash([]byte("ObsoleteRollupType(uint32)"))
	addNewRollupTypeSignatureHash                  = crypto.Keccak256Hash([]byte("AddNewRollupType(uint32,address,address,uint64,uint8,bytes32,string)"))

	// Events new ZkEvm/RollupBase
	acceptAdminRoleSignatureHash        = crypto.Keccak256Hash([]byte("AcceptAdminRole(address)"))                 // Used in oldZkEvm as well
	transferAdminRoleSignatureHash      = crypto.Keccak256Hash([]byte("TransferAdminRole(address)"))               // Used in oldZkEvm as well
	setForceBatchAddressSignatureHash   = crypto.Keccak256Hash([]byte("SetForceBatchAddress(address)"))            // Used in oldZkEvm as well
	setForceBatchTimeoutSignatureHash   = crypto.Keccak256Hash([]byte("SetForceBatchTimeout(uint64)"))             // Used in oldZkEvm as well
	setTrustedSequencerURLSignatureHash = crypto.Keccak256Hash([]byte("SetTrustedSequencerURL(string)"))           // Used in oldZkEvm as well
	setTrustedSequencerSignatureHash    = crypto.Keccak256Hash([]byte("SetTrustedSequencer(address)"))             // Used in oldZkEvm as well
	verifyBatchesSignatureHash          = crypto.Keccak256Hash([]byte("VerifyBatches(uint64,bytes32,address)"))    // Used in oldZkEvm as well
	sequenceForceBatchesSignatureHash   = crypto.Keccak256Hash([]byte("SequenceForceBatches(uint64)"))             // Used in oldZkEvm as well
	forceBatchSignatureHash             = crypto.Keccak256Hash([]byte("ForceBatch(uint64,bytes32,address,bytes)")) // Used in oldZkEvm as well
	sequenceBatchesSignatureHash        = crypto.Keccak256Hash([]byte("SequenceBatches(uint64,bytes32)"))          // Used in oldZkEvm as well
	initialSequenceBatchesSignatureHash = crypto.Keccak256Hash([]byte("InitialSequenceBatches(bytes,bytes32,address)"))
	updateEtrogSequenceSignatureHash    = crypto.Keccak256Hash([]byte("UpdateEtrogSequence(uint64,bytes,bytes32,address)"))

	// Extra RollupManager
	initializedSignatureHash               = crypto.Keccak256Hash([]byte("Initialized(uint64)"))                       // Initializable. Used in RollupBase as well
	roleAdminChangedSignatureHash          = crypto.Keccak256Hash([]byte("RoleAdminChanged(bytes32,bytes32,bytes32)")) // IAccessControlUpgradeable
	roleGrantedSignatureHash               = crypto.Keccak256Hash([]byte("RoleGranted(bytes32,address,address)"))      // IAccessControlUpgradeable
	roleRevokedSignatureHash               = crypto.Keccak256Hash([]byte("RoleRevoked(bytes32,address,address)"))      // IAccessControlUpgradeable
	emergencyStateActivatedSignatureHash   = crypto.Keccak256Hash([]byte("EmergencyStateActivated()"))                 // EmergencyManager. Used in oldZkEvm as well
	emergencyStateDeactivatedSignatureHash = crypto.Keccak256Hash([]byte("EmergencyStateDeactivated()"))               // EmergencyManager. Used in oldZkEvm as well

	// New GER event Etrog
	updateL1InfoTreeSignatureHash = crypto.Keccak256Hash([]byte("UpdateL1InfoTree(bytes32,bytes32)"))

	// PreLxLy events
	updateGlobalExitRootSignatureHash                   = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	preEtrogVerifyBatchesTrustedAggregatorSignatureHash = crypto.Keccak256Hash([]byte("VerifyBatchesTrustedAggregator(uint64,bytes32,address)"))
	transferOwnershipSignatureHash                      = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	updateZkEVMVersionSignatureHash                     = crypto.Keccak256Hash([]byte("UpdateZkEVMVersion(uint64,uint64,string)"))
	preEtrogConsolidatePendingStateSignatureHash        = crypto.Keccak256Hash([]byte("ConsolidatePendingState(uint64,bytes32,uint64)"))
	preEtrogOverridePendingStateSignatureHash           = crypto.Keccak256Hash([]byte("OverridePendingState(uint64,bytes32,address)"))
	sequenceBatchesPreEtrogSignatureHash                = crypto.Keccak256Hash([]byte("SequenceBatches(uint64)"))

	// Proxy events
	initializedProxySignatureHash = crypto.Keccak256Hash([]byte("Initialized(uint8)"))
	adminChangedSignatureHash     = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))
	beaconUpgradedSignatureHash   = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
	upgradedSignatureHash         = crypto.Keccak256Hash([]byte("Upgraded(address)"))
)

var scAddresses = []common.Address{
  common.HexToAddress(DefaultL1BridgeAddressSmartContract),
  common.HexToAddress(DefaultGlobalExitRootManager),
  common.HexToAddress(DefaultL1RollupManagerSmartContract),
  common.HexToAddress(DefaultL1ZkEVMSmartContract),
}

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

	zkevm, err := etrogpolygonzkevm.NewEtrogpolygonzkevm(common.HexToAddress(DefaultL1ZkEVMSmartContract), l1client)
  chkErr(err)

  rollupManager, err := etrogpolygonrollupmanager.NewEtrogpolygonrollupmanager(common.HexToAddress(DefaultL1RollupManagerSmartContract), l1client)
  chkErr(err)

  bridge, err := etrogpolygonzkevmbridge.NewEtrogpolygonzkevmbridge(common.HexToAddress(DefaultL1BridgeAddressSmartContract), l1client)
	chkErr(err)

  // Read currentBlock
	initBlock, err := l1client.BlockByNumber(ctx, nil)
	chkErr(err)
	initBlockNumber := initBlock.NumberU64()

  // Make a bridge tx
  amount := big.NewInt(1000000000000000)
	auth.Value = amount
	tx, err := bridge.BridgeAsset(auth, 1, auth.From, amount, common.Address{}, true, []byte{})
  chkErr(err)
  auth.Value = big.NewInt(0)

  err = operations.WaitTxToBeMined(ctx, l1client, tx, miningTimeout*time.Second)
  chkErr(err)

  // Now read the event
	finalBlock, err := l1client.BlockByNumber(ctx, nil)
	chkErr(err)
	finalBlockNumber := finalBlock.NumberU64()

	query := ethereum.FilterQuery{
    FromBlock: new(big.Int).SetUint64(initBlockNumber),
    ToBlock:   new(big.Int).SetUint64(finalBlockNumber),
    Addresses: scAddresses,
  }

  logs, err := l1client.FilterLogs(ctx, query)
  chkErr(err)

  for _, vLog := range logs {
    switch vLog.Topics[0] {
    	case updateL1InfoTreeSignatureHash:
    		log.Infof("UpdateL1InfoTree event detected: %v", vLog.Topics[0])
    	case depositEventSignatureHash:
		    log.Infof("Deposit event detected: %v", vLog.Topics[0])
		  default:
		    log.Infof("Event not registered: %+v", vLog.Topics[0])
    }
  }

  // Get tip
	tip, err := rollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	chkErr(err)
	log.Info("Tip: ", tip)

  rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"

	//txs := "ee80843b9aca00830186a0944d5cf5032b2a844602278b01199ed191a86c93ff88016345785d8a0000808203e980801186622d03b6b8da7cf111d1ccba5bb185c56deae6a322cebc6dda0556f3cb9700910c26408b64b51c5da36ba2f38ef55ba1cee719d5a6c012259687999074321bff"
  // Send forceBatch
	data := common.Hex2Bytes(rawTxs)
	log.Info("Data: ", data)
	tx, err = zkevm.ForceBatch(auth, data, tip)
	if err != nil {
		log.Error("error sending forceBatch. Error: ", err)
	}
	log.Info("TxHash: ", tx.Hash())

  err = operations.WaitTxToBeMined(ctx, l1client, tx, miningTimeout*time.Second)
	if err != nil {
		log.Error("error waiting tx to be mined. Error: ", err)
	}

  // Now read the event
	finalBlock, err = l1client.BlockByNumber(ctx, nil)
	chkErr(err)
	finalBlockNumber = finalBlock.NumberU64()

	query = ethereum.FilterQuery{
    FromBlock: new(big.Int).SetUint64(initBlockNumber),
    ToBlock:   new(big.Int).SetUint64(finalBlockNumber),
    Addresses: scAddresses,
  }

  logs, err = l1client.FilterLogs(ctx, query)
  chkErr(err)

//   	log.Infof("%v", setBatchFeeSignatureHash)
//   	log.Infof("%v", setTrustedAggregatorSignatureHash)
//   	log.Infof("%v", setVerifyBatchTimeTargetSignatureHash)
//   	log.Infof("%v", setMultiplierBatchFeeSignatureHash)
//   	log.Infof("%v", setPendingStateTimeoutSignatureHash)
//   	log.Infof("%v", setTrustedAggregatorTimeoutSignatureHash)
//   	log.Infof("%v", overridePendingStateSignatureHash)
//   	log.Infof("%v", proveNonDeterministicPendingStateSignatureHash)
//   	log.Infof("%v", consolidatePendingStateSignatureHash)
//   	log.Infof("%v", verifyBatchesTrustedAggregatorSignatureHash)
//   	log.Infof("%v", rollupManagerVerifyBatchesSignatureHash)
//   	log.Infof("%v", onSequenceBatchesSignatureHash)
//   	log.Infof("%v", updateRollupSignatureHash)
//   	log.Infof("%v", addExistingRollupSignatureHash)
//   	log.Infof("%v", createNewRollupSignatureHash)
//   	log.Infof("%v", obsoleteRollupTypeSignatureHash)
//   	log.Infof("%v", addNewRollupTypeSignatureHash)
//
//   	// Events new ZkEvm/RollupBase
//   	log.Infof("%v", acceptAdminRoleSignatureHash)
//   	log.Infof("%v", transferAdminRoleSignatureHash)
//   	log.Infof("%v", setForceBatchAddressSignatureHash)
//   	log.Infof("%v", setForceBatchTimeoutSignatureHash)
//   	log.Infof("%v", setTrustedSequencerURLSignatureHash)
//   	log.Infof("%v", setTrustedSequencerSignatureHash)
//   	log.Infof("%v", verifyBatchesSignatureHash)
//   	log.Infof("%v", sequenceForceBatchesSignatureHash)
//   	log.Infof("%v", forceBatchSignatureHash)
//   	log.Infof("%v", sequenceBatchesSignatureHash)
//   	log.Infof("%v", initialSequenceBatchesSignatureHash)
//   	log.Infof("%v", updateEtrogSequenceSignatureHash)
//
//   	// Extra RollupManager
//   	log.Infof("%v", initializedSignatureHash)
//   	log.Infof("%v", roleAdminChangedSignatureHash)
//   	log.Infof("%v", roleGrantedSignatureHash)
//   	log.Infof("%v", roleRevokedSignatureHash)
//   	log.Infof("%v", emergencyStateActivatedSignatureHash)
//   	log.Infof("%v", emergencyStateDeactivatedSignatureHash)
//
//   	// New GER event Etrog
//   	log.Infof("%v", updateL1InfoTreeSignatureHash)
//
//   	// PreLxLy events
//   	log.Infof("%v", updateGlobalExitRootSignatureHash)
//   	log.Infof("%v", preEtrogVerifyBatchesTrustedAggregatorSignatureHash)
//   	log.Infof("%v", transferOwnershipSignatureHash)
//   	log.Infof("%v", updateZkEVMVersionSignatureHash)
//   	log.Infof("%v", preEtrogConsolidatePendingStateSignatureHash)
//   	log.Infof("%v", preEtrogOverridePendingStateSignatureHash)
//   	log.Infof("%v", sequenceBatchesPreEtrogSignatureHash)
//
//   	// Proxy events
//   	log.Infof("%v", initializedProxySignatureHash)
//   	log.Infof("%v", adminChangedSignatureHash)
//   	log.Infof("%v", beaconUpgradedSignatureHash)
//   	log.Infof("%v", upgradedSignatureHash)

  for _, vLog := range logs {
    switch vLog.Topics[0] {
    	case updateL1InfoTreeSignatureHash:
    		log.Infof("UpdateL1InfoTree event detected: %v", vLog.Topics[0])
    	case depositEventSignatureHash:
		    log.Infof("Deposit event detected: %v", vLog.Topics[0])
		  case forceBatchSignatureHash:
		    log.Infof("ForceBatch event detected: %v", vLog.Topics[0])
      case onSequenceBatchesSignatureHash:
        log.Infof("OnSequenceBatches event detected: %v", vLog.Topics[0])
      case sequenceBatchesSignatureHash:
        log.Infof("SequenceBatches event detected: %v", vLog.Topics[0])
		  default:
		    log.Infof("Event not registered: %+v", vLog.Topics[0])
    }
  }

  var sequences []etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData
	sequences = append(sequences, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	}, etrogpolygonzkevm.PolygonRollupBaseEtrogBatchData{
		Transactions: common.Hex2Bytes(rawTxs),
	})
	_, err = zkevm.SequenceBatches(auth, sequences, uint64(time.Now().Unix()), uint64(1), auth.From)
	if err != nil {
  	log.Fatal(err)
  }
//
// 	// Mine the tx in a block
//   err = operations.WaitTxToBeMined(ctx, l1client, tx, miningTimeout*time.Second)
// 	if err != nil {
// 		log.Error("error waiting tx to be mined. Error: ", err)
// 	}
}

func chkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
