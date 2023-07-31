# Offchain protocol overview v2.1

This document aims to give a high level overview of a full e2e protocol for automation `v2.1`. 

<br />

## Table of Contents

  - [Overview](#overview)
  - [Boundaries](#boundaries)
  - [Definitions](#definitions)
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
- `upkeepTriggerID`: Uniquely identifies an attempt to do a unit of work and is represented as: `keccak256(upkeepID, abi.encode(trigger))` for evm chains.
- `upkeepPayload`: Input information to process an upkeep → (upkeepID, trigger, checkData)
    - For conditionals checkData is empty (derived onchain in checkUpkeep)
    - For log: checkData is log information
- `upkeepResult`: Output information to perform an upkeep. Same across both types: (fastGasWei, linkNative, upkeepID, trigger, gasLimit, performData)

## Upkeep Flows

### Conditional Upkeep Sampling/Eligibility Flow

### Conditional Upkeep Perform Flow

### Log Trigger Flow

### Log Recovery Flow

## [WIP] Visuals

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

The regsitry sync the active upkeeps and corresponding config onchain with node’s local state.

It does so by listening to the relevant events on chain for the following:
- Upkeep registration/cancellation
- Upkeep pause/unpause
- Upkeep config changes

Periodically (~10min) does a full sync of all upkeeps onchain state so that any potential missed / reorged logs are accounted for.

**Check upkeeps**

The registry exposes `checkUpkeep` so the runner could execute the pipeline:

- For log triggers verifies log is still part of chain at checkBlockNum
- Does a batch RPC call to checkUpkeep (for conditionals calls `checkUpkeep()` and for log calls `checkUpkeep(logData)`).
- Does batch mercury fetch and callback for upkeeps that requires mercury data
- Does batch RPC call to simulatePerformUpkeep for upkeeps that are eligible

Returns list of results for each payload. Result can be **eligible** with upkeepResult / **non-eligible** with failure reason / **error.** Error can be retryable if it’s transient or non retryable in case the checkBlockNum is too old.

### Runner

This component is responsible for parallelizing upkeep executions and providing a single interface to different components maintaining a shared cache

- Takes a list of upkeepPayloads, calls CheckPipeline asynchronously with upkeeps being batched in a single pipeline execution
- Maintains in-memory cache of non-errored pipeline executions indexed on (trigger, upkeepID). Directly uses that result instead of a new execution if available
- Allows for repeated calls for the same (trigger, upkeepID). If an execution is in progress then it doesn’t start a new execution
- Execution automatically fails after configured timeout. (~10s)
- Does a callback when results are available with the result

<aside>
💡 Note: This component can be made synchronous instead of giving a callback while callers can call it asynchronously
</aside>

### Coordinator

This component stores in-flight & confirmed reports in memory, and allows other components in the system to query that information. 
It maintains a single item per `upkeepTriggerID`, and stores `confirmed` flag, the `upkeepID` and `trigger`.

**Inflight reports**

The coordinator stores upkeep and trigger for a report which is inflight.

    - For conditionals only one inflight report should be present per upkeepID
    - For log upkeeps only one inflight report per (upkeepID, logIdentifier)

In case of conflict - a new report is seen for an upkeepID / (upkeepID, logIdentifier), it waits on the higher checkBlockNumber report.

Enables to mark as inflight with `Accept` function that is called from the `shouldAccept`.

**Transmit Events**

The coordinator listens to transmit event logs (stale / reorged / cancelled / insufficient funds) of stored `upkeepTriggerID` in memory, and act accordingly:

- Upkeep performed - the `upkeepTriggerID` is marked as confirmed.
- Stale report - the `upkeepTriggerID` is marked as confirmed.
    - conditional upkeeps will be sampled and checked again automatically
    - log upkeeps that were missed are expected to be picked up by the log recoverer **TBD**
- Reorged / insufficient funds - the `upkeepTriggerID` is removed from the inflight reports set. 
    - log upkeeps that were missed are expected to be picked up by the log recoverer

**TTL**

All keys stored expire after TTL. In case it was not marked as performed, it is assumed tx got lost.
- conditional upkeeps will be sampled and checked again automatically
- log upkeeps that were missed are expected to be picked up by the log recoverer

**Is Transmission Confirmed**

Provides additional read API to check whether some reported upkeep can be transmitted.
It expects the full trigger as input, i.e. with concrete checkBlockNum and checkBlockHash:
`isTransmissionConfirmed(upkeepID, trigger)`

**Should Process**

Provides additional read API to check whether an upkeep should be processed which is used for pre-processing / filtering to prevent duplicate reports. 
**Note that the input is a partially populated trigger which doesn’t include checkBlockNumber, checkBlockHash**. i.e.
    - For conditional: shouldProcess(upkeepID, blockNum)
        - False if report is inflight for upkeepID
        - False if report has been confirmed after blockNum (This can happen when network latest block is lagging this node’s logs)
        - True otherwise
    - For log: shouldProcess(upkeepID, logIdentifier)
        - False if report is inflight or has been confirmed
        - True otherwise

### Result Store

This component is responsible for storing upkeepResults that a node thinks should be performed. It hopes to get agreement from quorum of nodes on these results to push into reports. Best effort is made to ensure the same logs enter different node’s result stores independently as **it is assumed that blockchain nodes will get in sync within TTL**, but it is not guaranteed as node’s local blockchain can see different reorgs or select different logs during surges. For results that do not achieve agreement within TTL go into recovery flow.

- Maintains an in-memory collection of upkeepResults. It should maintain a single result per `upkeepTriggerID`. 
    - Overwrites results for duplicated `upkeepTriggerID`.
- Each result has a TTL. When the TTL expires
    - conditional upkeeps will be sampled and checked again automatically
    - log upkeeps that were missed are expected to be picked up by the log recoverer **TBD**
- Provides a read API to view results, which takes as input (`pseudoRandomSeed`, `limit`). Sorts all keys (trigger, upkeepID) with the `seed` and provides first `limit` results. We do not do FIFO here as potentially out of sync old results will block new results sent by the node for agreement.
- Provides an API to remove (trigger, upkeepID) from the store

### Metadata Store

This component stores the metadata to be sent within observations to the network. There are three categories of metadata

- Conditional Upkeep Instructions
- Log recovery Instructions
- Latest block history

Every item has a TTL, and upon expiry items are just removed without any action.

The store provides an add / view / remove API for other components.

### Samples Observer

Processes samples of upkeeps to be checked, using the latest block number as a trigger. It does the following procedures:

- Pre-processes to filter upkeep present in coordinator
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into **metadata store**
with conditional Upkeep instructions
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

<aside>
💡 Note: Duplicate logs can enter this flow, this component should be able to handle that. This component can maintain in-memory state for processed logs and can also use that state for exponential backoff
</aside>

### Retry Observer

**TODO**

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
- It will start scanning from `lastRePollBlock` on each iteration, and update the block number when it finishes scanning.
- Logs that are older than 24hr are ignored, therefore `lastRePollBlock` starts at `latestBlock - (24hr block)` in case it was not populated before.

#### Upkeep States

The upkeeps states are used to track the status of log upkeeps (eligible, performed) across the system, to avoid redundant work by the recoverer. Enables to select by (upkeepID, logIdentifier) is used as a key to store the state of a log upkeep.

The state is updated by the coordinator when the upkeep is performed or after positive check (eligible).

The states will be persisted to so the latest state to be restored when the node starts up.

### Plugin

The plugin is performing the following tasks upon OCR3 procedures:

#### Observation

Observation starts with a processing of the previous outcome:
- Remove agreed upon finalized results from result store
- For `acceptedSamples` (bound to a trigger)
    - Remove from metadata store
    - Enqueue (trigger, upkeepID) into conditional observer
- For `acceptedRecoveryLogs` (bound to a trigger)
    - Remove from metadata store
    - Enqueue (trigger, upkeepID) into recovery observer

Then we do the following for current observation:
- Query results from result store giving seqNr as pseudoRandom seed and predefined limit. Filter results using coordinator and add them to observation
- Query from metadata store for sample and recovery instructions using seqNr as pseudoRandom seed within limits, filter using coordinator and add them to observation
- Query latest block number and hashes from local chain, add them to observation

#### Outcome

- Derive latest blockNumber and blockHash
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
