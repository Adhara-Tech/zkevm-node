package main

/*
error BatchAlreadyVerified()
0x812a372d

error BatchNotSequencedOrNotSequenceEnd()
0x98c5c014

error ExceedMaxVerifyBatches()
0xb59f753a

error FinalNumBatchBelowLastVerifiedBatch()
0xb9b18f57

error FinalNumBatchDoesNotMatchPendingState()
0x32a2a77f

error FinalPendingStateNumInvalid()
0xbfa7079f

error ForceBatchNotAllowed()
0x24eff8c3

error ForceBatchTimeoutNotExpired()
0xc44a0821

error ForceBatchesAlreadyActive()
0xf6ba91a1

error ForceBatchesDecentralized()
0xc89374d8

error ForceBatchesNotAllowedOnEmergencyState()
0x39258d18

error ForceBatchesOverflow()
0xc630a00d

error ForcedDataDoesNotMatch()
0xce3d755e

error GasTokenNetworkMustBeZeroOnEther()
0x1a874c12

error GlobalExitRootNotExist()
0x73bd668d

error HaltTimeoutNotExpired()
0xd257555a

error HaltTimeoutNotExpiredAfterEmergencyState()
0x3d49ed4c

error HugeTokenMetadataNotSupported()
0x248b8f82

error InitNumBatchAboveLastVerifiedBatch()
0x1e56e9e2

error InitNumBatchDoesNotMatchPendingState()
0x2bd2e3e7

error InitSequencedBatchDoesNotMatch()
0x1a070d9a

error InvalidInitializeTransaction()
0xcd161966

error InvalidProof()
0x09bde339

error InvalidRangeBatchTimeTarget()
0xe067dfe8

error InvalidRangeForceBatchTimeout()
0xf5e37f2f

error InvalidRangeMultiplierBatchFee()
0x4c2533c8

error MaxTimestampSequenceInvalid()
0x0a00feb3

error NewAccInputHashDoesNotExist()
0x66385b51

error NewPendingStateTimeoutMustBeLower()
0x48a05a90

error NewStateRootNotInsidePrime()
0x176b913c

error NewTrustedAggregatorTimeoutMustBeLower()
0x401636df

error NotEnoughMaticAmount()
0x4732fdb5

error NotEnoughPOLAmount()
0x2354600f

error OldAccInputHashDoesNotExist()
0x6818c29e

error OldStateRootDoesNotExist()
0x4997b986

error OnlyAdmin()
0x47556579

error OnlyPendingAdmin()
0xd1ec4b23

error OnlyRollupManager()
0xb9b3a2c8

error OnlyTrustedAggregator()
0xbbcbbc05

error OnlyTrustedSequencer()
0x11e7be15

error PendingStateDoesNotExist()
0xbb14c205

error PendingStateInvalid()
0xd086b70b

error PendingStateNotConsolidable()
0x0ce9e4a2

error PendingStateTimeoutExceedHaltAggregationTimeout()
0xcc965070

error SequenceWithDataAvailabilityNotAllowed()
0x821935b4

error SequenceZeroBatches()
0xcb591a5f

error SequencedTimestampBelowForcedTimestamp()
0x7f7ab872

error SequencedTimestampInvalid()
0xea827916

error StoredRootMustBeDifferentThanNewRoot()
0xa47276bd

error SwitchToSameValue()
0x5f0e7abe

error TransactionsLengthAboveMax()
0xa29a6c7c

error TrustedAggregatorTimeoutExceedHaltAggregationTimeout()
0x1d06e879

error TrustedAggregatorTimeoutNotExpired()
0x8a0704d3

event AcceptAdminRole(address newAdmin)
0x056dc487bbf0795d0bbb1b4f0af523a855503cff740bfb4d5475f7a90c091e8e

event ForceBatch(uint64 indexed forceBatchNum, bytes32 lastGlobalExitRoot, address sequencer, bytes transactions)
0xf94bb37db835f1ab585ee00041849a09b12cd081d77fa15ca070757619cbc931

event InitialSequenceBatches(bytes transactions, bytes32 lastGlobalExitRoot, address sequencer)
0x060116213bcbf54ca19fd649dc84b59ab2bbd200ab199770e4d923e222a28e7f

event Initialized(uint8 version)
0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498

event SequenceBatches(uint64 indexed numBatch, bytes32 l1InfoRoot)
0x3e54d0825ed78523037d00a81759237eb436ce774bd546993ee67a1b67b6e766

event SequenceForceBatches(uint64 indexed numBatch)
0x648a61dd2438f072f5a1960939abd30f37aea80d2e94c9792ad142d3e0a490a4

event SetDataAvailabilityProtocol(address newDataAvailabilityProtocol)
0xd331bd4c4cd1afecb94a225184bded161ff3213624ba4fb58c4f30c5a861144a

event SetForceBatchAddress(address newForceBatchAddress)
0x5fbd7dd171301c4a1611a84aac4ba86d119478560557755f7927595b082634fb

event SetForceBatchTimeout(uint64 newforceBatchTimeout)
0xa7eb6cb8a613eb4e8bddc1ac3d61ec6cf10898760f0b187bcca794c6ca6fa40b

event SetTrustedSequencer(address newTrustedSequencer)
0xf54144f9611984021529f814a1cb6a41e22c58351510a0d9f7e822618abb9cc0

event SetTrustedSequencerURL(string newTrustedSequencerURL)
0x6b8f723a4c7a5335cafae8a598a0aa0301be1387c037dccc085b62add6448b20

event SwitchSequenceWithDataAvailability()
0xf32a0473f809a720a4f8af1e50d353f1caf7452030626fdaac4273f5e6587f41

event TransferAdminRole(address newPendingAdmin)
0xa5b56b7906fd0a20e3f35120dd8343db1e12e037a6c90111c7e42885e82a1ce6

event VerifyBatches(uint64 indexed numBatch, bytes32 stateRoot, address indexed aggregator)
0x9c72852172521097ba7e1482e6b44b351323df0155f97f4ea18fcec28e1f5966


function GLOBAL_EXIT_ROOT_MANAGER_L2() view returns (address)
0x9e001877

function INITIALIZE_TX_BRIDGE_LIST_LEN_LEN() view returns (uint8)
0x11e892d4

function INITIALIZE_TX_BRIDGE_PARAMS() view returns (bytes)
0x05835f37

function INITIALIZE_TX_BRIDGE_PARAMS_AFTER_BRIDGE_ADDRESS() view returns (bytes)
0x7a5460c5

function INITIALIZE_TX_BRIDGE_PARAMS_AFTER_BRIDGE_ADDRESS_EMPTY_METADATA() view returns (bytes)
0x52bdeb6d

function INITIALIZE_TX_CONSTANT_BYTES() view returns (uint16)
0x03508963

function INITIALIZE_TX_CONSTANT_BYTES_EMPTY_METADATA() view returns (uint16)
0x676870d2

function INITIALIZE_TX_DATA_LEN_EMPTY_METADATA() view returns (uint8)
0xc7fffd4b

function INITIALIZE_TX_EFFECTIVE_PERCENTAGE() view returns (bytes1)
0x40b5de6c

function SIGNATURE_INITIALIZE_TX_R() view returns (bytes32)
0xb0afe154

function SIGNATURE_INITIALIZE_TX_S() view returns (bytes32)
0xd7bc90ff

function SIGNATURE_INITIALIZE_TX_V() view returns (uint8)
0xf35dda47

function TIMESTAMP_RANGE() view returns (uint256)
0x42308fab

function acceptAdminRole()
0x8c3d7301

function admin() view returns (address)
0xf851a440

function bridgeAddress() view returns (address)
0xa3c573eb

function calculatePolPerForceBatch() view returns (uint256)
0x00d0295d

function dataAvailabilityProtocol() view returns (address)
0xe57a0b4c

function forceBatch(bytes transactions, uint256 polAmount)
0xeaeb077b

function forceBatchAddress() view returns (address)
0x2c111c06

function forceBatchTimeout() view returns (uint64)
0xc754c7ed

function forcedBatches(uint64) view returns (bytes32)
0x6b8616ce

function gasTokenAddress() view returns (address)
0x3c351e10

function gasTokenNetwork() view returns (uint32)
0x3cbc795b

function generateInitializeTransaction(uint32 networkID, address _gasTokenAddress, uint32 _gasTokenNetwork, bytes _gasTokenMetadata) view returns (bytes)
0xa652f26c

function globalExitRootManager() view returns (address)
0xd02103ca

function initialize(address _admin, address sequencer, uint32 networkID, address _gasTokenAddress, string sequencerURL, string _networkName)
0x71257022

function initializeMigration()
0x1c8b9370

function isSequenceWithDataAvailabilityAllowed() view returns (bool)
0x4c21fef3

function lastAccInputHash() view returns (bytes32)
0x6e05d2cd

function lastForceBatch() view returns (uint64)
0xe7a7ed02

function lastForceBatchSequenced() view returns (uint64)
0x45605267

function networkName() view returns (string)
0x107bf28c

function onVerifyBatches(uint64 lastVerifiedBatch, bytes32 newStateRoot, address aggregator)
0x32c2d153

function pendingAdmin() view returns (address)
0x26782247

function pol() view returns (address)
0xe46761c4

function rollupManager() view returns (address)
0x49b7b802

function sequenceBatches((bytes transactions, bytes32 forcedGlobalExitRoot, uint64 forcedTimestamp, bytes32 forcedBlockHashL1)[] batches, uint64 maxSequenceTimestamp, uint64 initSequencedBatch, address l2Coinbase)
0xdef57e54

function sequenceBatchesValidium((bytes32 transactionsHash, bytes32 forcedGlobalExitRoot, uint64 forcedTimestamp, bytes32 forcedBlockHashL1)[] batches, uint64 maxSequenceTimestamp, uint64 initSequencedBatch, address l2Coinbase, bytes dataAvailabilityMessage)
0xdb5b0ed7

function sequenceForceBatches((bytes transactions, bytes32 forcedGlobalExitRoot, uint64 forcedTimestamp, bytes32 forcedBlockHashL1)[] batches)
0x9f26f840

function setDataAvailabilityProtocol(address newDataAvailabilityProtocol)
0x7cd76b8b

function setForceBatchAddress(address newForceBatchAddress)
0x91cafe32

function setForceBatchTimeout(uint64 newforceBatchTimeout)
0x4e487706

function setTrustedSequencer(address newTrustedSequencer)
0x6ff512cc

function setTrustedSequencerURL(string newTrustedSequencerURL)
0xc89e42df

function switchSequenceWithDataAvailability(bool newIsSequenceWithDataAvailabilityAllowed)
0x2acdc2b6

function transferAdminRole(address newPendingAdmin)
0xada8f919

function trustedSequencer() view returns (address)
0xcfa8ed47

function trustedSequencerURL() view returns (string)
0x542028d5
*/
