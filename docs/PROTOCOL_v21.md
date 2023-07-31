# Offchain protocol overview v2.1

This document aims to give a high level overview of a full e2e protocol for automation `v2.1`. 

<br />

## Table of Contents

  - [Overview](#overview)
  - [Boundaries](#boundaries)
  - [Definitions](#definitions)
  - [Upkeep Flows](#upkeep-flows)
    - [Sampling Flow](#sampling-flow)
    - [Perform Flow](#perform-flow)
    - [Log Trigger Flow](#log-trigger-flow)
    - [Log Recovery Flow](#log-recovery-flow)
  - [Visuals](#wip-visuals)
  - [Components](#components)
    - Common:
        - [Registry](#registry)
        - [Upkeep Index (Life Cycle)](#upkeep-index-life-cycle)
        - [Runner](#runner)
        - [Coordinator](#coordinator)   
        - [Result Store](#result-store)
        - [Metadata Store](#metadata-store)
    - Conditional Triggers:
        - [Samples Observer](#samples-observer)
        - [Conditional Observer](#conditional-observer)
    - Log Triggers:
        - [Log Provider](#log-provider)
            - [Log Buffer](#log-buffer)
        - [Log Recoverer](#log-recoverer)
        - [Log Observer](#log-observer)
        - [Recovery Observer](#recovery-observer)
        - [Log Recoverer](#log-recoverer)
        - [Upkeep States](#upkeep-states)
    - [OCR3 Plugin](#plugin)
        - [Observation](#observation)
        - [Outcome](#outcome)
        - [Reports](#reports)
        - [ShouldAcceptFinalizedReport](#shouldacceptfinalizedreport)
        - [ShouldTransmitAcceptedReport](#shouldtransmitacceptedreport)

<br />

## Overview

The idea behind the protocol is to provide a decentralized execution engine to automate smart contract interaction, with a general infrastructure to support future triggers from other sources.

## Boundaries

The protocol works with n=10 nodes, handling upto f=2 arbitrary malicious nodes. It aims to give the following guarantees:

### Reliability Guarantees

**Log triggers** 
Out of n=10 nodes every node listens to configured user log, as soon f+1=3 nodes see the log and agree on checkPipeline, it will be performed on chain. We can handle up to 7 nodes missing a log and handle capacity of upto 10 log trigger upkeeps with rate limit per upkeep of 5 logs per second.

**Conditionals** 
Every upkeep’s condition will be checked at least once by the network every ~3 seconds, handling up to f+1=3 nodes being down. Once condition is eligible, every node evaluates the checkPipeline, as soon as f+1=3 nodes agree, it will be performed on chain. We can handle capacity of upto 500 conditional upkeeps.
    
The protocol will be functional as long as > 6 ((n+f)/2) nodes are alive and participating within the p2p network (required for ocr3 consensus).

### Security Guarantees

At least f+1=3 independent nodes need to achieve agreement on an upkeep, trigger and its data before it is performed.


## Definitions

- `upkeepID`: Unique 256 bit identifier for an upkeep. Each upkeep has a unique trigger type (conditional or log) which is encoded within the ID
- `trigger`: Used to represent the trigger for a particular upkeep performance
    - For conditionals: (checkBlockNum, checkBlockHash)
    - For log triggers: (checkBlockNum, checkBlockHash, logTxHash, logIndex)
- `logIdentifier`: unique identifier for a log → (logTxHash, logIndex)
- `upkeepPayload`: Input information to process a unit of work for an upkeep → (upkeepID, trigger, checkData)
    - For conditionals checkData is empty (derived onchain in checkUpkeep)
    - For log: checkData is log information
- `upkeepTriggerID`: Uniquely identifies an `upkeepPayload` and is represented as: `keccak256(upkeepID, abi.encode(trigger))` for evm chains.
- `upkeepResult`: Output information to perform an upkeep. Same across both types: (fastGasWei, linkNative, upkeepID, trigger, gasLimit, performData)

## Upkeep Flows

The eligibility flows are the sequence of events and procedures used to determine if an upkeep is considered eligible to perform.

The protocol supports two types of triggers:

### 1. Conditional Triggers Flows

A trigger that is based on a block number and block hash. It is used to trigger an upkeep based on a condition that is evaluated on-chain.

#### Sampling Flow

The sampling flow is used to determine if an upkeep is eligible to perform. It is
triggered by a ticker that provides samples of upkeeps to check. The samples are
collected, filtered, and checked. The results are then pushed into the metadata store with the corresponding instructions. 
The plugin will then collect the instructions and push them into the outcome to be processed in next round.

Supported instructions:
- Coordinated samples - are sample upkeeps that are eligible to perform
  - The upkeeps are performed with an assoicated block
- Recovered logs - are logs that we identify as missed and need to be recovered
  - The logs are performed with an associated block

#### Perform Flow


### 2. Log Triggers

#### Log Trigger Flow

#### Log Recovery Flow

## Visuals

Source of truth here: https://miro.com/app/board/uXjVPntyh4E=/

Conditional triggers:

![Conditional Triggers Diagram](./images/block_ocr3_conditional_triggers.png)

Log triggers:

![Log Triggers Diagram](./images/block_ocr3_log_triggers.png)

## Components

### Registry

This is the main component that connects the offchain node with the registry onchain.

The registry offers the following functionality:

**Upkeeps life-cycle management**

The regsitry sync the active upkeeps and corresponding config onchain with node’s local state in memory.

It does so by listening to the relevant events on chain for the following:
- Upkeep registration/cancellation/migration
- Upkeep pause/unpause
- Upkeep config changes

Periodically (~10min) does a full sync of all upkeeps onchain state so that any potential missed / reorged state logs are accounted for.

The registry calls the log event provider in case some log filter needs to be updated.
Upon startup, all upkeeps are synced from chain and the log event provider is called to update all the log filters.

**Check upkeeps**

The registry exposes `checkUpkeep` so the runner could execute the pipeline taking `upkeepPayload` as an input

- If the checkBlockNum is lagging latestBlock by more than a threshold (100 blocks) then it returns a non-retryable error
- For log triggers verifies the log is still part of chain at checkBlockNum
- Does a batch RPC call to checkUpkeep (for conditionals calls `checkUpkeep()` and for log calls `checkUpkeep(logData)`).
- Does batch mercury fetch and callback for upkeeps that requires mercury data
- Does batch RPC call to simulatePerformUpkeep for upkeeps that are eligible

Returns list of results for each payload. Result can be **eligible** with upkeepResult / **non-eligible** with failure reason / **error**, which can be retryable if it’s transient or non retryable e.g. in case the checkBlockNum is too old.

### Runner

This component is responsible for parallelizing upkeep executions and providing a single interface to different components maintaining a shared cache

- Takes a list of upkeepPayloads, calls CheckPipeline asynchronously with upkeeps being batched in a single pipeline execution
- Maintains in-memory cache of non-errored pipeline executions indexed on (trigger, upkeepID). Directly uses that result instead of a new execution if available
- Allows for repeated calls for the same upkeep payload
- Execution automatically fails after a timeout that was provided as argument (~10s, set by Observer)
- A call to CheckUpkeeps on the runner is synchronous. A worker is spawned per a batch of upkeeps to check, and all workers needs to finish before returning.

<aside>
Note: Because of the sync nature, we don't track pending requests, so there might be double checking of same payloads. Once a payload was checked, we cache the result in memory, so next time we don't need to check it again.
</aside>

### Coordinator

This component stores in-flight & confirmed reports in memory, and allows other components in the system to query that information.

The coordinator stores inflight reports, in case of conflict - a new report is seen for the same key, it waits on the higher `checkBlockNumber`. The `Accept` function is called from the `shouldAccept` on the plugin.

All keys stored expire after TTL. In case an item is not marked as performed within TTL, it is assumed that the tx got lost. Conditionals and Log upkeeps recover from such scenarios through different logic.

#### Conditional Coordinator

The coordinator for conditional triggers ensures that an upkeep is performed at progressively higher blocks and tracks 'in-flight' status on transmits. This variant sets the last performed block higher in a local cache for an upkeepId on every transmit event.

The coordinator stores inflight reports per `upkeepID` and an associated `upkeepTriggerID` for identifying transmit events.

#### Log Trigger Coordinator

The coordinator for log triggers ensures that a triggered log is performed exactly once. It tracks 'in-flight' status when a report is generated and registers transmit events such as performed or staleReport or reorg.

The coordinator stores inflight reports per `(upkeepID, logIdentifier)` and an associated `upkeepTriggerID` for identifying transmit events.


**Transmit Events**

The coordinator listens to transmit event logs (stale / reorged / cancelled / insufficient funds) of stored `upkeepTriggerID` in memory, and act accordingly:

- Upkeep performed - the `upkeepTriggerID` is marked as confirmed
    - For log upkeeps, state is updated to `performed` with the Upkeep States component
    - For conditional upkeeps the perform block number is stored in memory
- Stale report - the `upkeepTriggerID` is marked as confirmed
    - conditional upkeeps will be sampled and checked again automatically on a later block
    - for log upkeeps it means there was a duplicate perform. The state is updated to `performed` in Upkeep States
- Reorged - the `upkeepTriggerID` is removed from coordinator
    - conditional upkeeps will be sampled and checked again automatically on a later block
    - Logs enqueued through log event provider (i.e. checkBlockNum == logBlockNum) are forgotten about since the log got reorged. If reorg causes a new log then it is expected to be surfaced up by the log provider again.
    - Logs enqueued through recovery attemtps will be picked up again by the log recoverer
- Insufficient funds - the `upkeepTriggerID` is removed
    - This can happen when we thought there were sufficient funds during checkPipeline but during onchain execution funds were not enough (e.g. gas price spike)
    - conditional upkeeps will be sampled and checked again automatically on a later block
    - Logs are expected to be picked up again by the log recoverer
- Cancelled Upkeep - the `upkeepTriggerID` is removed. No recovery is needed as upkeep is cancelled

**Is Transmission Confirmed**

Provides additional read API to check whether some reported upkeep can be transmitted.
It expects the full trigger as input, i.e. with concrete checkBlockNum and checkBlockHash:
`isTransmissionConfirmed(upkeepID, trigger)`.

An item is considered confirmed if:
- was not seen yet, or removed by the coordinator (e.g. reorged)
- marked as confirmed by the coordinator

**Should Process**

Provides additional read API to check whether an upkeep should be processed which is used for pre-processing / filtering to prevent duplicate reports.
    - For conditional: shouldProcess(upkeepID, blockNum)
        - False if report is inflight for upkeepID
        - False if report has been confirmed after blockNum (This can happen when network latest block is lagging this node’s logs)
        - True otherwise
    - For log: shouldProcess(upkeepID, logIdentifier)
        - False if report is inflight or has been confirmed
        - True otherwise

### Result Store

This component is responsible for storing `upkeepResults` that a node thinks should be performed. It hopes to get agreement from quorum of nodes on these results to push into reports. Best effort is made to ensure the same logs enter different node’s result stores independently as **it is assumed that blockchain nodes will get in sync within TTL**, but it is not guaranteed as node’s local blockchain can see different reorgs or select different logs during surges. Results that do not achieve agreement within TTL are expected to be picked up by recovery flow.

- Maintains an in-memory collection of upkeepResults
    - Conditional upkeeps will be stored by (upkeepID)
    - Log upkeeps will be stored by (upkeepID, logIdentifier)
    - Overwrites results for duplicated results with newer checkBlockNumber.
- Each result has a TTL (5min). When the TTL expires
    - conditional upkeeps will be sampled and checked again automatically
    - log upkeeps that were missed are expected to be picked up by the log recoverer
- Provides a read API to view results, which takes as input (`pseudoRandomSeed`, `limit`).
The nodes in the network are expected to use the same seed for a particular round. Sorts all keys (trigger, upkeepID) with the `seed` and provides first `limit` results. 
Note: We do not do FIFO here as potentially out of sync old results will block new results sent by the node for agreement.
- Provides an API to remove (trigger, upkeepID) from the store

### Metadata Store

This component stores the metadata to be sent within observations to the network. There are three categories of metadata

- Conditional Upkeep instructions (`eligible samples`)
- Log recovery instructions (`recovery logs`)
- Latest block history

Every item has a TTL, and upon expiry items are just removed without any action.

The store provides an add / view / remove API for other components.

### Samples Observer

The sampler ticker calls the samples observer every second, with samples of upkeeps to checked. It uses the latest block number as the trigger. It does the following procedures:

- Pre-processes to filter upkeep present in coordinator
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into **metadata store**
with coordinated sample instruction
- Errors and ineligible results are ignored

### Conditional Observer

Processes coordinated block + upkeep payload coming from plugin and does the following:

- Pre-processes to filter upkeep present in coordinator
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into **result store**
- Errors and ineligible results are ignored

### Log Observer

Processes latest logs for active upkeeps. It does the following procedures:

- Called every second by Log Trigger Ticker with latest logs as full payloads.
- Filter out logs with block number that is older than threshold
    - These are expected to be picked up by the log recoverer 
- Pre-processes to filter the logs already present in coordinator
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into result store
- If runner gave an error, put back into log retry ticker, to be performed after a fixed timeout (can extend to exponential backoff)

### Retry Observer

Allows to retry log upkeeps with scheduled retry timing. 

### Recovery Observer

This flow gets as input an upkeepID and a logIdentifier. It is a backup flow which verifies that a log was fully processed. Logs can be put here from the node itself (through Log Recoverer) or from other nodes (through Plugin)

- Input is an upkeepID, logTxHash, logIndex (optional checkBlockNum, checkBlockHash when passed from plugin)
- Checks whether log should be processed
    - Whether its part of the filters for that upkeep and it is still present on chain (DB call to log poller)
    - Whether it is older from the latest block within a threshold (To prevent malicious node requesting recovery on a new log)
- Checks whether the log has been already processed
    - If node locally checked (logTxHash, logIndex) and got a non eligible result then skip (DB call)
    - If chain has a perform for the log then skip (DB call which looks at upkeepPerformed logs)
- If checkBlockNum not present: puts into an **in-memory recovery queue** which is then read in the plugin and put back in the flow with a coordinated checkBlock
- If checkBlockNum present
    - get payload from log provider for (upkeepID, logTxHash, logIndex)
    - call runner with payload
    - upon result, if eligible add into result store
    - non-eligible are written to DB
    - If runner gave an error then it is ignored (it might get picked up again by log recoverer, but we do not keep retrying here)

<aside>
💡 Note: Duplicate logs can enter this flow, this component should be able to handle that
</aside>

### Log Provider

This component’s purpose is to surface latest logs for registered upkeeps. It does not maintain any state across restarts (no DB). The main functionality it exposes

- Listening for log filter config changes from Upkeep Index and syncing log poller with the filters
- Provides a simple interface `getLatestLogs` to provide new **unseen** logs across all upkeeps within a limit
    - Repeatedly queries latest logs from the chain (via log poller DB) for the last `lookbackBlocks` (~200) blocks. Stores them in the log buffer (see below)
    - Handles load balancing and rate limiting across upkeeps
    - `getLatestLogs` limits the number of logs returned per upkeep in a single call. If there are more logs present, then the provider gives the logs starting from an offset (`latestBlock-latestBlock%lookbackBlocks`). Offset is calculated such that the nodes try to choose the same logs from big pool of logs so they can get agreement

<aside>
💡 Note: `getLatestLogs` might miss logs when there is a surge of logs which lasts longer than `lookbackBlocks`. Upon node restarts it can **miss logs** when it restarts after a gap, or **provide same logs again** when it quickly restarts
</aside>

- Provides an interface `getPayloadLog` to build an upkeep payload for a particular log on demand by giving a trigger as in input.
It gets only the required log from the log buffer or reads it from log poller if not found in buffer. This is used by the recovery flow

#### Log Buffer

A circular/ring buffer of blocks and their corresponding logs, that act as a cache for logs. 
It is used to store the last `lookbackBlocks*3` blocks, where each block holds a list of that block's logs. 
Logs are marked as seen when they are returned by the buffer, to avoid working with logs that have already been seen.

It is used by the log provider to provide unknown logs to the node, and by by the log recoverer to identify known logs during recovery flow.


### Log Recoverer

The log recoverer is responsible to ensure that no logs are missed.
It does that by running a background process for re-scanning of logs and putting the ones we missed into the recovery flow (without checkBlockNum/Hash).

Logs will be considered as missed if they are older than `latestBlock - lookbackBlocks` and has not been performed or successfully checked already (eligible).

While the provider is scanning latest logs, the recoverer is scanning older logs in ascending order, up to `latestBlock - lookbackBlocks`, newer blocks will be under the provider's lookback window.

**Recoverer scanning process**

- The recoverer maintains a `lastRePollBlock` for each upkeep, .i.e. the last block it scanned for that upkeep.
- Every second, the recoverer will scan logs for a subset of `n=10` upkeeps, where `n/2` upkeeps are randomly chosen and `n/2` upkeeps are chosen by the oldest `lastRePollBlock`.
- It will start scanning from `lastRePollBlock` on each iteration
- Logs that are older than 24hr are ignored, therefore `lastRePollBlock` starts at `latestBlock - (24hr block)` in case it was not populated before.
- `lastRePollBlock` is updated in case there are no logs in a specific range, otherwise will wait for performed events to know that all logs in that range were processed before updating `lastRePollBlock`.

#### Upkeep States

The upkeeps states are used to track the status of log upkeeps (eligible, performed) across the system, to avoid redundant work by the recoverer. Enables to select by (upkeepID, logIdentifier) is used as a key to store the state of a log upkeep.

The state is updated by the coordinator when the upkeep is performed or after positive check (eligible).

The states will be persisted to so the latest state to be restored when the node starts up.

### Plugin

The plugin is performing the following tasks upon OCR3 procedures:

#### Observation

Observation starts with a processing of the previous outcome:
- Remove agreed upon finalized results from result store
- For `acceptedSamples` (bound to a trigger - latest coordinated block from the previous outcome)
    - Remove from metadata store
    - Enqueue (trigger, upkeepID) into conditional observer
- For `acceptedRecoveryLogs` (bound to a trigger - latest coordinated block from the previous outcome)
    - Remove from metadata store
    - Enqueue (trigger, upkeepID) into recovery observer

Then we do the following for current observation:
- Query results from result store giving seqNr as pseudoRandom seed and predefined limit. Filter results using coordinator and add them to observation
- Query from metadata store for sample and recovery instructions using seqNr as pseudoRandom seed within limits, filter using coordinator and add them to observation
- Query latest block number and hashes from local chain, add them to observation

#### Outcome

- Derive latest blockNumber and blockHash, by looking on block history and using the most recent block/hash that the majority of nodes have in common
- Any result which has f+1 agreement is added to finalized result
- All samples are collected from observations within limits, deduped and filtered from existing `acceptedSamples`. These are then added to `acceptedSamples` in the outcome bound to the current latestBlockNumber and hash.
    - `acceptedSamples` is a ring buffer where samples are held for ‘x’ (~30) rounds so that they do not get bound to a new blockNumber for some time
- Similar behaviour as samples is done for recovery logs to maintain `acceptedRecoveryLogs`
 
#### Reports

Takes finalised results from the outcome, package them into reports with potential batching of upkeeps.
Batching is subject to upkeep gas limit, and preconfigured reportGasLimit and gasOverheadPerUpkeep. Additionally, same upkeep ID is not batched within the same report.

For a list of upkeepResults, we only need to send one fastGasWei, linkNative to chain in the report. This is taken from the result which has the highest checkBlockNum

#### ShouldAcceptFinalizedReport

Extracts [(trigger, upkeepID)] from report and adds reported upkeeps to the coordinator to be marked as inflight. Will return always true.

<aside>
💡 Note: We cannot guarantee that the same (upkeepID) / (logIdentifier, upkeepID) will not be already existing in coordinator. (e.g. node’s local chain is lagging the network). We need to have an override behaviour where we wait on the higher checkBlockNum report.
</aside>

#### ShouldTransmitAcceptedReport

Extracts [(trigger, upkeepID)] from report filters upkeeps that were already performed using the coordinator. If any (trigger, upkeepID) is not yet confirmed then return true
