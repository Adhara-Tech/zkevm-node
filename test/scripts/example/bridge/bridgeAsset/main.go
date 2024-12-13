package main

import (
  "context"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmbridge"
  "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/etrogpolygonzkevmglobalexitroot"
  "github.com/0xPolygonHermez/zkevm-node/log"
  "github.com/0xPolygonHermez/zkevm-node/test/operations"
  "github.com/ethereum/go-ethereum"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/ethclient"
  "math/big"
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

const (
  DefaultL1NetworkURL        = "http://localhost:8545"
  DefaultL1ChainID    uint64 = 1337
  DefaultL2NetworkURL        = "http://localhost:8123"
  DefaultL2ChainID    uint64 = 1001

  DefaultDeployerAddress     = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  DefaultDeployerPrivateKey  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  DefaultSequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"
  DefaultSequencerPrivateKey = "0x28b2b0318721be8c8339199172cd7cc8f5e273800a35616ec893083a4b32c02e"

  DefaultL1BridgeSmartContract        = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
  DefaultL1GERManagerSmartContract    = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
  DefaultL1ZkEVMSmartContract         = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
  DefaultL1RollupManagerSmartContract = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
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

  initBlock, err := client.BlockByNumber(ctx, nil)
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

  // Check events
  log.Debug("Reading Events..........")
  finalBlock, err := client.BlockByNumber(ctx, nil)
  if err != nil {
	log.Fatal(err)
  }
  // Filter query
  query := ethereum.FilterQuery{
	FromBlock: new(big.Int).SetUint64(initBlock.NumberU64()),
	ToBlock:   new(big.Int).SetUint64(finalBlock.NumberU64()),
	Addresses: []common.Address{common.HexToAddress(DefaultL1ZkEVMSmartContract), common.HexToAddress(DefaultL1RollupManagerSmartContract), common.HexToAddress(DefaultL1GERManagerSmartContract)},
  }
  logs, err := client.FilterLogs(ctx, query)
  if err != nil {
	log.Fatal(err)
  }
  for _, vLog := range logs {
	switch vLog.Topics[0] {
	case sequenceBatchesSignatureHash:
	  log.Debug("sequenceBatchesSignatureHash", vLog.Address)
	case sequenceBatchesPreEtrogSignatureHash:
	  log.Debug("sequenceBatchesPreEtrogSignatureHash", vLog.Address)
	case updateGlobalExitRootSignatureHash:
	  log.Debug("updateGlobalExitRootSignatureHash", vLog.Address)
	case updateL1InfoTreeSignatureHash:
	  log.Debug("updateL1InfoTreeSignatureHash", vLog.Address)
	case forceBatchSignatureHash:
	  log.Debug("forceBatchSignatureHash", vLog.Address)
	case initialSequenceBatchesSignatureHash:
	  log.Debug("initialSequenceBatchesSignatureHash", vLog.Address)
	case updateEtrogSequenceSignatureHash:
	  log.Debug("initialSequenceBatchesSignatureHash", vLog.Address)
	case verifyBatchesTrustedAggregatorSignatureHash:
	  log.Debug("VerifyBatchesTrustedAggregator event detected. Ignoring...", vLog.Address)
	case rollupManagerVerifyBatchesSignatureHash:
	  log.Debug("RollupManagerVerifyBatches event detected. Ignoring...", vLog.Address)
	case preEtrogVerifyBatchesTrustedAggregatorSignatureHash:
	  log.Debug("preEtrogVerifyBatchesTrustedAggregatorSignatureHash", vLog.Address)
	case verifyBatchesSignatureHash:
	  log.Debug("verifyBatchesSignatureHash", vLog.Address)
	case sequenceForceBatchesSignatureHash:
	  log.Debug("sequenceForceBatchesSignatureHash", vLog.Address)
	case setTrustedSequencerURLSignatureHash:
	  log.Debug("SetTrustedSequencerURL event detected. Ignoring...", vLog.Address)
	case setTrustedSequencerSignatureHash:
	  log.Debug("SetTrustedSequencer event detected. Ignoring...", vLog.Address)
	case initializedSignatureHash:
	  log.Debug("Initialized event detected. Ignoring...", vLog.Address)
	case initializedProxySignatureHash:
	  log.Debug("InitializedProxy event detected. Ignoring...", vLog.Address)
	case adminChangedSignatureHash:
	  log.Debug("AdminChanged event detected. Ignoring...", vLog.Address)
	case beaconUpgradedSignatureHash:
	  log.Debug("BeaconUpgraded event detected. Ignoring...", vLog.Address)
	case upgradedSignatureHash:
	  log.Debug("Upgraded event detected. Ignoring...", vLog.Address)
	case transferOwnershipSignatureHash:
	  log.Debug("TransferOwnership event detected. Ignoring...", vLog.Address)
	case emergencyStateActivatedSignatureHash:
	  log.Debug("EmergencyStateActivated event detected. Ignoring...", vLog.Address)
	case emergencyStateDeactivatedSignatureHash:
	  log.Debug("EmergencyStateDeactivated event detected. Ignoring...", vLog.Address)
	case updateZkEVMVersionSignatureHash:
	  log.Debug("updateZkEVMVersionSignatureHash", vLog.Address)
	case consolidatePendingStateSignatureHash:
	  log.Debug("ConsolidatePendingState event detected. Ignoring...", vLog.Address)
	case preEtrogConsolidatePendingStateSignatureHash:
	  log.Debug("PreEtrogConsolidatePendingState event detected. Ignoring...", vLog.Address)
	case setTrustedAggregatorTimeoutSignatureHash:
	  log.Debug("SetTrustedAggregatorTimeout event detected. Ignoring...", vLog.Address)
	case setTrustedAggregatorSignatureHash:
	  log.Debug("SetTrustedAggregator event detected. Ignoring...", vLog.Address)
	case setPendingStateTimeoutSignatureHash:
	  log.Debug("SetPendingStateTimeout event detected. Ignoring...", vLog.Address)
	case setMultiplierBatchFeeSignatureHash:
	  log.Debug("SetMultiplierBatchFee event detected. Ignoring...", vLog.Address)
	case setVerifyBatchTimeTargetSignatureHash:
	  log.Debug("SetVerifyBatchTimeTarget event detected. Ignoring...", vLog.Address)
	case setForceBatchTimeoutSignatureHash:
	  log.Debug("SetForceBatchTimeout event detected. Ignoring...", vLog.Address)
	case setForceBatchAddressSignatureHash:
	  log.Debug("SetForceBatchAddress event detected. Ignoring...", vLog.Address)
	case transferAdminRoleSignatureHash:
	  log.Debug("TransferAdminRole event detected. Ignoring...", vLog.Address)
	case acceptAdminRoleSignatureHash:
	  log.Debug("AcceptAdminRole event detected. Ignoring...", vLog.Address)
	case proveNonDeterministicPendingStateSignatureHash:
	  log.Debug("ProveNonDeterministicPendingState event detected. Ignoring...", vLog.Address)
	case overridePendingStateSignatureHash:
	  log.Debug("OverridePendingState event detected. Ignoring...", vLog.Address)
	case preEtrogOverridePendingStateSignatureHash:
	  log.Debug("PreEtrogOverridePendingState event detected. Ignoring...", vLog.Address)
	case roleAdminChangedSignatureHash:
	  log.Debug("RoleAdminChanged event detected. Ignoring...", vLog.Address)
	case roleGrantedSignatureHash:
	  log.Debug("RoleGranted event detected. Ignoring...", vLog.Address)
	case roleRevokedSignatureHash:
	  log.Debug("RoleRevoked event detected. Ignoring...", vLog.Address)
	case onSequenceBatchesSignatureHash:
	  log.Debug("OnSequenceBatches event detected. Ignoring...", vLog.Address)
	case updateRollupSignatureHash:
	  log.Debug("updateRollupSignatureHash", vLog.Address)
	case addExistingRollupSignatureHash:
	  log.Debug("addExistingRollupSignatureHash", vLog.Address)
	case createNewRollupSignatureHash:
	  log.Debug("createNewRollupSignatureHash", vLog.Address)
	case obsoleteRollupTypeSignatureHash:
	  log.Debug("ObsoleteRollupType event detected. Ignoring...", vLog.Address)
	case addNewRollupTypeSignatureHash:
	  log.Debug("addNewRollupType event detected but not implemented. Ignoring...", vLog.Address)
	case setBatchFeeSignatureHash:
	  log.Debug("SetBatchFee event detected. Ignoring...", vLog.Address)
	}
	log.Debug("Event not registered: ", vLog.Topics[0], vLog.Address)
	//log.Debugf("Event not registered: %+v", vLog)
  }

}
