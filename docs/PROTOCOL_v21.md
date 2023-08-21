# Offchain protocol overview v2.1

This document aims to give a high level overview of a full e2e protocol for automation `v2.1`. 

## Table of Contents

  - [Overview](#overview)
  - [Boundaries](#boundaries)
  - [Definitions](#definitions)
  - [Eligibility Flows](#eligibility-flows)
    - [Conditional Triggers Flows](#1-conditional-triggers-flows)
        - [Conditional Proposal](#conditional-proposal-flow)
        - [Conditional Finalization](#conditional-finalization-flow)
    - [Log Triggers](#2-log-triggers)
        - [Log Trigger](#log-trigger-flow)
        - [Retry](#retry-flow)
        - [Log Recovery Proposal](#log-recovery-proposal-flow)
        - [Log Recovery Finalization](#log-recovery-finalization-flow)
  - [Visuals](#visuals)
  - [Components](#components)
    - Common:
        - [Registry](#registry)
        - [Runner](#runner)
        - [Transmit Event Provider](#transmit-event-provider)
        - [Coordinator](#coordinator)   
        - [Result Store](#result-store)
        - [Metadata Store](#metadata-store)
        - [Block Ticker](#block-ticker)
    - Conditional Triggers:
        - [Samples Observer](#samples-observer)
        - [Conditional Observer](#conditional-observer)
    - Log Triggers:
        - [Log Observer](#log-observer)
        - [Retry Observer](#retry-observer)
        - [Recovery Observer](#recovery-observer)
        - [Log Provider](#log-provider)
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
- `trigger`: Used to represent the trigger for a particular upkeep performance, and is represented as: `(checkBlockNum, checkBlockHash,extension)` where the extension is based on the trigger type:
    - Conditionals: no extension 
    - Log triggers: `(logTxHash, logIndex, logBlockNum, logBlockHash)`. \
    NOTE: `logBlockNum` and `logBlockHash` might not be present in the trigger, in which case they are set to 0 and empty respectively. In such cases the log block will be resolved the given tx hash.
- `logIdentifier`: unique identifier for a log → `(logTxHash, logIndex)`
- `workID`: Unique 256 bit identifier for a unit of work that is used across the system. \
`(upkeepID, trigger)` are used to form a workID, in different structure, based on the trigger type:
    - Conditionals: `keccak256(upkeepID)`. Where we allow sequential execution of the same upkeepID, in cases the trigger has a newer `checkBlockNum`, higher then the last performed check block.
    - Log triggers: `keccak256(upkeepID,logIdentifier)`. At any point in time there can be at most one unit of work for a particular upkeep and log.
- `upkeepPayload`: Input information to process a unit of work for an upkeep → `(upkeepID, trigger, checkData)`
    - For conditionals checkData is empty (derived onchain in checkUpkeep)
    - For log: checkData is log information
- `upkeepResult`: Output information to perform an upkeep. Same across both types: `(fastGasWei, linkNative, upkeepID, trigger, gasLimit, performData)`

## Eligibility Flows

The eligibility flows are the sequence of events and procedures used to determine if an upkeep is considered eligible to perform.

The protocol supports two types of triggers:

### 1. Conditional Triggers Flows
#### Conditional Proposal Flow

The sampling flow is used to determine if an upkeep is eligible to perform. It is
triggered by a ticker that provides samples of upkeeps to check. The samples are
collected, filtered, and checked. The results are then pushed into the metadata store as proposals. 
The plugin will then collect these proposals and push them into the outcome to be processed in next rounds, where they will go into conditional finalization flow.

<aside>
A node can be temporarily down and miss some rounds and associated actions on outcome. A ring buffer of coordinated proposals is kept for 20 rounds. A node can process coorindated proposals for upto last 20 rounds.

`1sec` is the expected OCR round time, so the timeout for a coordinated proposal is `~20sec`.
It gives the observe enough time to process the proposal before it gets coordinated again, on a new block number. 
</aside>

#### Conditional Finalization Flow 

The conditional finalization flow is used to come to agreement among nodes on what upkeepPayloads to check, based on the results of the proposal flow. It is triggered by a ticker that provides payloads based on a coordinated block and upkeepIDs.

The results are collected, filtered, and checked again. Eligible results will go into the results store and later on into a report and those that were agreed by at least f+1=3 nodes will be performed on chain.

### 2. Log Triggers
#### Log Trigger Flow

The log trigger flow is used to determine if a log needs to be perform. It is triggered by a ticker that get the latest logs from log event provider.
The payloads are filtered, processed through checkPipeline and eligible results are collected into the result store. Those that are agreed by at least f+1=3 nodes will go into a report and be performed on chain.

In cases of retryable failures, the payloads are pushed into the retry queue.

#### Retry Flow

The retry flow is used to retry payloads that failed with retryable errors. It is triggered by a ticker that gets payloads from the retry queue.

The payloads are filtered, processed through checkPipeline and eligible results are collected into the result store. Those that are agreed by at least f+1=3 nodes will go into a report and be performed on chain.

#### Log Recovery Proposal Flow

The log recovery flow is used to recover logs that were missed by the log trigger flow. It is triggered by a ticker that gets missed logs from log recoverer.
The missed logs are pushed into the metadata store as recovery proposals. 
The plugin will then collect these proposals and push them into the outcome to be processed in next rounds where they gets picked up into recovery finalization flow. 

<aside>
A node can be temporarily down and miss some rounds and associated actions on outcome. A ring buffer of coordinated proposals is kept for 20 rounds. A node can process coorindated proposals for upto last 20 rounds.

`1sec` is the expected OCR round time, so the timeout for a coordinated proposal is `~20sec`.
It gives the observe enough time to process the proposal before it gets coordinated again, on a new block number. 
</aside>

#### Log Recovery Finalization Flow

The recovery finalization flow takes recoverable payloads merged with the latest check blocks and runs the pipeline for them.

The recovery finalization ticker will call the payload builder to build payloads with the latest logs. 
The log recoverer does necessary checks to ensure that the log should actually be recovered, to protect against malicious nodes surfacing wrong logs for recovery. 
The payloads will then go into log observer to be checked again. 
Eligible results will go into the results store and later on into a report and those 
that were agreed by at least f+1=3 nodes will be performed on chain.

## Visuals

The diagrams below shows the data flow between components. The diagrams are simplified to show only the relevant components for each trigger and the corresponding flows.

💡 Note: source is available [here](https://miro.com/app/board/uXjVPntyh4E=/).

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

**Check pipline**

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
💡 Note: Because of the sync nature, we don't track pending requests, so there might be double checking of same payloads. Once a payload was checked, we cache the result in memory, so next time we don't need to check it again.
</aside>

### Transmit Event Provider

This component serves to transmit events from log poller, to other components in the system (coordinator). Transmit events are the events that can happen when a report is sent onchain to the contract:

- `UpkeepPerformed`: Report successfully performed for an upkeep 
(for log triggers `(upkeepID, logIdentifier)`)
- `StateUpkeep`: Report was stale and the upkeep was already performed
    - For conditionals this happens when an upkeep is tried to be performed on a checkBockNumber which is older than the last perform block (Stale check). 
    - For log triggers this happens when the particular (upkeepID, logIdentifier) has been performed before already
- `InsufficientFunds`: Emitted when pre upkeep execution when not enough funds are found on chain for the execution. Funds check is done in checkPipeline, but actual funds required on chain at execution time can change, e.g. to gas price changes / link price changes. In such cases upkeep is not performed or charged. These reports should really be an edge case, on chain we have a multiplier during checkPipeline to overestimate funds before even attempting an upkeep.
- `CancelledUpkeep`: This happens when the upkeep gets cancelled in between check time and perform time. To protect against malicious users, the contract adds a 50 block delay to any user cancellation requests.

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

This component stores the metadata to be sent within observations to the network. 
Every record has a TTL, and upon expiry items are just removed without any action.
The store provides an add / view / remove API for other components.

There are three categories of objects that are stored:

**1. Conditional proposals**

Are sample upkeeps (upkeepID) that are eligible to perform.

**2. Log recovery proposals**

Are recovered proposals (upkeepID, logIdentifier) that were missed, and are eligible to perform.

**3. Latest block history**

Latest block history coming from the block subscriber (last 256 block and hashes), and used for coordination among nodes.

## Observers

### Proposal Observers

Proposal observers takes input from the corresponding ticker, process it and push it into the metadata store to be incloded as proposals for the next round.

Proposals will be added to the propsal queue, be processed by the finalization observers.

There are 2 types of proposal observers:

#### 1. Samples Observer
Manages samples of conditional upkeeps, gets sample payloads from the upkeep provider.

#### 2. Recovery Observer
Manages recovery proposals of log upkeeps, gets recovery payloads from the log recoverer.

<br />

Proposal observers will perform the following steps:

**Pre-processing**

Filters upkeeps present in coordinator (`shouldProcess`).

**Processing**

Calls runner with upkeep payloads

**Post-processing**

Does the following:

- For eligible upkeeps, adds them to the metadata store (`eligible samples` / `recovery logs`)
- For ineligible upkeeps, updates only log trigger state in upkeep states store to be ineligible
- Retryables and other errors are ignored
    - For conditionals, they will be anyway picked up in next samples.
    - For log triggers, we expect the recoverer to pick them up again if needed


### Finalization Observers

Finalization observers takes input from the corresponding ticker, process it and push it into the result store to be to be included in reports by the plugin.

There are 4 types of finalization observers:

#### 1. Conditional Finalization
Processes agreed eligible samples from previous observations.

#### 2. Log Trigger
Processes log triggers that were found by log provider.

#### 3. Log Recovery Finalization
Processes agreed eligible recovery logs from previous observations.

#### 4. Retry
Processes retryable payloads from the retry queue.

<br />

Finalization observers will perform the following steps:

**Pre-processing**

Filters upkeeps present in coordinator (`shouldAccept`).

**Processing**

Calls runner with upkeep payloads.

**Post-processing**

Does the following:

- For eligible upkeeps, adds them to the result store
- For ineligible upkeeps, updates only log trigger state in upkeep states store to be ineligible
- Retryables errors are added to the retry queue

### Log Provider

This component’s purpose is to surface latest logs for registered upkeeps. It does not maintain any state across restarts (no DB). The main functionality it exposes

- Listening for log filter config changes from Registry: 
    - sync log poller with the filters
    - sync the local filter store with changes in filters
- Provides a simple interface `getLatestPayloads` to provide new **unseen** logs across all upkeeps within a limit
    - Repeatedly queries latest logs from the chain (via log poller DB) for the last `lookbackBlocks` (200) blocks. Stores them in the log buffer (see below)
    - Handles load balancing and rate limiting across upkeeps
    - `getLatestPayloads` limits the number of logs returned per upkeep in a single call. If there are more logs present, then the provider gives the logs starting from an offset (`latestBlock-latestBlock%lookbackBlocks`). Offset is calculated such that the nodes try to choose the same logs from big pool of logs so they can get agreement

<aside>
💡 Note: `getLatestPayloads` might miss logs when there is a surge of logs which lasts longer than `lookbackBlocks`. Upon node restarts it can **miss logs** when it restarts after a gap, or **provide same logs again** when it quickly restarts
</aside>

#### Log Buffer

A circular/ring buffer of blocks and their corresponding logs, that act as a cache for logs. It is used to store the last `lookbackBlocks` blocks, where each block holds a list of that block's logs.

Logs are marked as seen when they are returned by the buffer, to avoid working with logs that have already been seen.

It is used by the log provider to provide unknown logs to the node, and by by the log recoverer to identify known logs during recovery flow.

#### Log Filter Store

A local store of log filters for each upkeep. It is used by the log provider and log recoverer as a source of truth for the current active log filters.

### Log Recoverer

The log recoverer is responsible to ensure that no logs are missed.
It does that by running a background process for re-scanning of logs and putting the ones we missed into the recovery flow (without checkBlockNum/Hash).

Logs will be considered as missed if they are older than `latestBlock - lookbackBlocks` and has not been performed or successfully checked already (ineligible result).

While the provider is scanning latest logs, the recoverer is scanning older logs in ascending order, up to `latestBlock - lookbackBlocks`, newer blocks will be under the provider's lookback window.

**Recoverer scanning process**

- The recoverer maintains a `lastRePollBlock` for each upkeep, i.e. the last block it scanned for that upkeep.
- Every second, the recoverer will scan logs for a subset of `n=10` upkeeps, where `n/2` upkeeps are randomly chosen and `n/2` upkeeps are chosen by the oldest `lastRePollBlock`.
- It will start scanning from `lastRePollBlock` on each iteration
- Logs that are older than 24hr are ignored, therefore `lastRePollBlock` starts at `latestBlock - (24hr block)` in case it was not populated before.
- `lastRePollBlock` is updated in case there are no logs in a specific range, otherwise will wait for performed events to know that all logs in that range were processed before updating `lastRePollBlock`.

**Proposal Data**

The recoverer provides an interface `getProposalData` to be called when buildubg an upkeep payload for a particular log on demand by giving a trigger as in input.

The payload building process does the following:
- Does not return a payload for a newer log to ensure recovery logs are isolated from latest logs. 
- Verifies that the log is part of the filters for that upkeep and it is still present on chain
- Checks whether the log has been already processed within the upkeepState. If so then doesn't return a payload
- Gets the required log from log poller
- Packs the log for the payload

### Upkeep States

The upkeeps states are used to track the status of log upkeeps (ineligible, performed) across the system, to avoid redundant work by the recoverer. Enables to select by (upkeepID, logIdentifier) is used as a key to store the state of a log upkeep.

The state is updated after ineligible check by observer.
Perform events scanner updates the states cache lazily/on-demand, by reading `DedupKeyAdded` events.

The states (only ineligible) will be persisted in DB so the latest state to be restored when the node starts up.

<aside>
💡 Note: Performed states are not persisted in DB, as they are already present in log events that are stored in DB.
</aside>

<aside>
💡 Note: Using a DB might introduce inconsistencies across the nodes in the network, e.g. in case of node restarts.
</aside>

### Plugin

The plugin is performing the following tasks upon OCR3 procedures:

#### Observation

Observation starts with a processing of the previous outcome:
- Remove agreed upon finalized results from result store
- For `acceptedSamples` (already bound to a trigger - latest coordinated block from the previous outcome)
    - Remove `upkeepID` from metadata store
    - Enqueue (trigger, upkeepID) into coordinated ticker
- For `acceptedRecoveryLogs` (already bound to a trigger - latest coordinated block from the previous outcome)
    - Remove from metadata store
    - Enqueue (trigger, upkeepID) into recovery finalization ticker

Then we do the following for current observation:
- Query results from result store giving seqNr as pseudoRandom seed and predefined limit. Filter results using coordinator and add them to observation
- Query from metadata store for conditional samples and recovery instructions within limits, filter using coordinator and add them to observation
- Query block history and hashes from metadata store, add them to observation

#### Outcome

- Derive latest blockNumber and blockHash, by looking on block history and using the most recent block/hash that the majority of nodes have in common. It is not added to the outcome automatically but is used below
- Any result which has f+1 agreement is added to finalized result
- All conditional samples are collected from observations within limits, deduped and filtered from existing `acceptedSamples`. These are then added to `acceptedSamples` in the outcome **bound to the current latestBlockNumber and hash**.
    - `acceptedSamples` is a ring buffer where samples are held for ~30 rounds so that they get deduped and not get bound to a new blockNumber for some rounds
- Similar behaviour as conditional samples is done for recovery logs to maintain `acceptedRecoveryLogs`

#### Reports

Takes finalised results from the outcome, package them into reports with potential batching of upkeeps.
Batching is subject to upkeep gas limit, and preconfigured reportGasLimit and gasOverheadPerUpkeep. Additionally, same upkeep ID is not batched within the same report.

For a list of upkeepResults, we only need to send one fastGasWei, linkNative to chain in the report. This is taken from the result which has the highest checkBlockNum

#### ShouldAcceptFinalizedReport

Extracts [(trigger, upkeepID)] from report and adds reported upkeeps to the coordinator to be marked as inflight. Will return always true.

<aside>
💡 Note: We cannot guarantee that the same (upkeepID) / (logIdentifier, upkeepID) will not be already existing in coordinator. (e.g. node’s local chain is lagging the network). As a result we have an override behaviour where we wait on the higher checkBlockNum report.
</aside>

#### ShouldTransmitAcceptedReport

Extracts [(trigger, upkeepID)] from report filters upkeeps that were already performed using the coordinator. If any (trigger, upkeepID) is not yet confirmed then return true
