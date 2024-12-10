# Title

The title of the research topic or experiment.

## Introduction

The LxLy bridge is a central component to the AggLayer which offers multi-chain interoperability.

The LxLy bridge currently works with the Polygon zkEVM as the L2 and the Ethereum network as L1.

The LxLy bridge is an interoperability solution aimed at enabling cross-chain communication among networks. It facilitates interaction between two L2 chains or between an L2 chain and Ethereum as the L1.

The zkEVM bridge’s architecture consists mainly of three (3) smart contracts:

- The bridge contract (PolygonZkEVMBridge.sol), which handles transfer of assets and messages between networks.
- The global exit root manager contract (PolygonZkEVMGlobalExitRoot.sol), which facilitates synchronization of state-info between the L2 and the L1.
- The Polygon zkEVM consensus contract (PolygonZkEVMEtrog.sol), which handles the sequencing and verification of transactions in the form of batches.

### Global exit trees

Critical to the design of the LxLy bridge are exit trees and the global exit tree.

Each chain has a Merkle tree called an exit tree, to which a leaf containing data of each asset-transfer is appended. Since such a leaf records data of the asset exiting the chain, it is called an Exit Leaf.

Another Merkle tree whose leaves are roots of the various exit trees is formed, and it is called the global exit tree.

The root of the global exit tree is the single source of state-truth communicated between rollups.

It is the global exit root manager contract’s responsibility to update the global exit tree root and acts as a custodian for the global exit tree’s history.

A complete transfer of assets in version-1 involves invoking three smart contracts; PolygonZkEVMEtrog.sol, PolygonZkEVMBridge.sol and PolygonZkEVMGlobalExitRoot.sol.

In the context of the LxLy bridge, the rollup manager contract verifies sequenced batches from various networks.

Consensus contracts of each connected network handle the sequencing of their own batches, but send the batch data to the rollup manager contract for verification.

The rollup manager contract stores the information of the sequenced batches in the form of an accumulated input hash, as in the version-1 of the zkEVM bridge.

Once sequenced batches have been verified, the global exit tree gets updated, in an approach similar to the zkEVM bridge version-1.

### Overall flow of events
The following diagram captures the following flow of events, most of which are handled by the rollup manager contract:

- Updating rollup manager’s lists.
- Creating rollups.
- Sequencing of batches.
- Aggregation or proving of batches.
- Verification of batches.
- Updating the global exit root.

## Problem Description

A motivation of why this problem is interesting, what the problem is and a brief description of what we intend to do to better assess the problem.

## Methodology

A detailed guide for setting up the experiment and/or how the problem is solved.

## Evaluation

A guide to running tests against the experiment and/or some information around how to assess the solution. This section may also include a results table or outputs obtained from the testing process.

## Main Findings and Conclusions

A summary of what was done/achieved.

## Future Work

(Optional) A look to next steps that are suggested by the author.

## References

(Optional) A list of resources that was referenced, which may be research papers providing background information or other previously implemented experiments performed by adhara-labs. 
