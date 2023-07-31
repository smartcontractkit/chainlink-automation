# Offchain protocol overview v2.1

This document aims to give a high level overview of a full e2e protocol for automation `v2.1`. 

## Table of Contents

  - [Overview](#overview)
  - [Boundaries](#boundaries)
  - [Definitions](#definitions)
  - [Upkeep Flows](#upkeep-flows)
    - [Sampling Flow](#sampling-flow)
    - [Perform Flow](#perform-flow)
    - [Log Trigger Flow](#log-trigger-flow)
    - [Log Recovery Flow](#log-recovery-flow)
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
        - [Log Provider](#log-provider)
            - [Log Buffer](#log-buffer)
        - [Log Recoverer](#log-recoverer)
        - [Log Observer](#log-observer)
        - [Retry Observer](#retry-observer)
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
#### Sampling Flow

The sampling flow is used to determine if an upkeep is eligible to perform. It is
triggered by a ticker that provides samples of upkeeps to check. The samples are
collected, filtered, and checked. The results are then pushed into the metadata store with `EligibleSample` [instruction](#instructions). 
The plugin will then collect the instructions and push them into the outcome to be processed in next rounds, where they will go into coordination flow.

<aside>
A node can be temporarily down and miss some rounds and associated actions on outcome. A ring buffer of coordinated upkeeps is kept for 30 rounds. A node can process coorindated upkeeps for upto last 30 rounds.
</aside>

#### Coordination Flow

The coordination flow is used to come to agreement among nodes on what upkeepPayloads to check, based on the results of the sampling flow. It is triggered by a ticker that provides
payloads based on a coordinated block and upkeepIDs. 
The results are collected, filtered, and checked again. Eligible results will go into the results store and later on into a report and those that were agreed by at least f+1=3 nodes will be performed on chain.

### 2. Log Triggers
#### Log Trigger Flow

The log trigger flow is used to determine if a log needs to be perform. It is triggered by a ticker that get the latest logs from log event provider.
The payloads are filtered, processed through checkPipeline and eligible results are collected into the result store. Those that are agreed by at least f+1=3 nodes will go into a report and be performed on chain.

In cases of retryable failures, the payloads are scheduled to be retried into the retry ticker.

#### Log Recovery Proposal Flow

The log recovery flow is used to recover logs that were missed by the log trigger flow. It is triggered by a ticker that gets missed logs from log recoverer.
The missed logs are pushed into the metadata store with `RecoveredLog` [instruction](#instructions). 
The plugin will then collect the instructions and push them into the outcome to be processed in next rounds where they gets picked up into recovery finalization flow. 

<aside>
Similar to coordinated upkeeps in conditional flow, A node can be temporarily down and miss some rounds and associated actions on outcome. A ring buffer of RecoveredLogs is kept for 30 rounds. A node can process recovered logs for upto last 30 rounds.
</aside>

#### Log Recovery Finalization Flow

The recovery finalization flow takes recoverable payloads merged with the latest check blocks and runs the pipeline for them.

The recovery finalization ticker will call log provider to build payloads with the latest logs. The log provider does necessary checks to ensure that the log should actually be recovered (To protect against malicious nodes surfacing wrong logs for recovery). The payloads will then go into log observer to be checked again. Eligible results will go into the results store and later on into a report and those that were agreed by at least f+1=3 nodes will be performed on chain.

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
💡 Note: Because of the sync nature, we don't track pending requests, so there might be double checking of same payloads. Once a payload was checked, we cache the result in memory, so next time we don't need to check it again.
</aside>

### Transmit Event Provider

This component listens to transmit events from log poller. Upon seeing new events calls the coordinator. Transmit events are the events that can happen when a report is sent onchain to the contract. These are

- UpkeepPerformed: Report successfully performed the upkeep it was meant for. It was the `upkeepTriggerId` within the log to identify the payload which performed the upkeep
- StateUpkeep: For conditionals this happens when an upkeep is tried to be performed on a checkBockNumber which is older than the last perform block (Stale check). For logs this happens when the particular (upkeepID, logIdentifier) has been performed before
- InsufficientFunds: This happens when pre upkeep execution when not enough funds are found on chain for the execution. Funds check is done in checkPipeline, but actual funds required on chain at execution time can change, e.g. to gas price changes / link price changes. In such cases upkeep is not performed or charged. These reports should really be an edge case, on chain we have a multiplier during checkPipeline to overestimate funds before even attempting an upkeep.
- CancelledUpkeep: This happens when the upkeep gets cancelled in between check time and perform time. To protect against malicious users, the contract adds a 50 block delay to any user cancellation requests.


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

#### Instructions

**Eligible samples** 
are sample upkeeps (upkeepID) that are eligible to perform.
**handling:** the upkeeps are bound to an associated block in the outcome to be processed in future rounds.

**Recovered logs**
are logs that we identify as missed and need to be recovered.
**handling:** the logs are bound to an associated block in the outcome to be processed in future rounds.

### Block Ticker

Powered by the block subscriber. On every tick, it fetches latest block history from the block subscriber (last 256 block and hashes) and updates them in the metadata store.

### Samples Observer

Powered by the sample ticker which calls the samples observer on every tick, with samples of upkeeps [taken from registry] to checked. It uses the latest block number as the trigger. It does the following procedures:

- Pre-processes to filter upkeeps present in coordinator (Using `shouldProcess`)
- Calls runner with upkeep payload
- If upkeep is eligible add `upkeepID` into **metadata store** (`eligible samples`)
- Errors and ineligible results are ignored

### Conditional Observer

Powered by the coordinated ticker. Coordinated ticker allows for coordinated `upkeepID` + `checkBlockNum/checkBlockHash` to be given as input by the plugin. It stores them in memory and on every tick calls the conditional observer with inputs till that time. 

Observer does the following:
- Pre-processes to filter upkeep present in coordinator (Using `shouldProcess`)
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into **result store**
- Errors and ineligible results are ignored

### Log Observer

Powered by the log trigger ticker and recovery finalization ticker. 
- Log trigger ticker: on every tick it calls `getLatestPayloads` from the Log Provider and calls observer with the payloads.
- Log recovery finalization ticker: Recovery ticker fetches full `upkeepPayloads` for the coordinated logs given as input within the tick and calls observer with the payloads.

For all payloads, observer does the following:
- Pre-processes to filter the logs already present in coordinator
- Calls runner with upkeep payload
- If upkeep is eligible enqueue into result store
- If runner gave an ineligible result, update log state in upkeep state to be ineligble
- If runner gave a retryable error, put into scheduled retry ticker which will be scheduled for retry
- If runner gave a non-retryable error, ignore. Will be picked up by log recoverer.

### Retry Observer

Powered by the scheduled retry ticker. The ticker manages scheduling of the retryable payloads, on every tick calls the observer with the payloads that should be retried.

The rest of the flow of the observer is the same as Log Observer.

### Recovery Observer

This observer is powered by the recovery ticker. Recovery ticker gets log recovery proposals `(upkeepID, logIdentifier)` which are not tied to a coordinated check block.

On every tick
- Calls `getMissedLogs` and surfaces recovery proposals to the recovery Observer.

Observer does the following:
- Puts the recovery proposals into **metadata store** (`recovery logs`)

### Log Provider

This component’s purpose is to surface latest logs for registered upkeeps. It does not maintain any state across restarts (no DB). The main functionality it exposes

- Listening for log filter config changes from Registry and sync log poller with the filters
- Provides a simple interface `getLatestLogs` to provide new **unseen** logs across all upkeeps within a limit
    - Repeatedly queries latest logs from the chain (via log poller DB) for the last `lookbackBlocks` (200) blocks. Stores them in the log buffer (see below)
    - Handles load balancing and rate limiting across upkeeps
    - `getLatestLogs` limits the number of logs returned per upkeep in a single call. If there are more logs present, then the provider gives the logs starting from an offset (`latestBlock-latestBlock%lookbackBlocks`). Offset is calculated such that the nodes try to choose the same logs from big pool of logs so they can get agreement

<aside>
💡 Note: `getLatestLogs` might miss logs when there is a surge of logs which lasts longer than `lookbackBlocks`. Upon node restarts it can **miss logs** when it restarts after a gap, or **provide same logs again** when it quickly restarts
</aside>

- Provides an interface `getPayloadLog` to build an upkeep payload for a particular log on demand by giving a trigger as in input.
- It does not return a payload for a newer log to ensure recovery logs are isolated from latest logs. It verifies that the log is part of the filters for that upkeep and it is still present on chain
- It checks whether the log has been already processed within the upkeepState. If so then doesn't return a payload
- It gets the required log from log poller. This is used by the recovery flow


#### Log Buffer

A circular/ring buffer of blocks and their corresponding logs, that act as a cache for logs. It is used to store the last `lookbackBlocks` blocks, where each block holds a list of that block's logs.

Logs are marked as seen when they are returned by the buffer, to avoid working with logs that have already been seen.

It is used by the log provider to provide unknown logs to the node, and by by the log recoverer to identify known logs during recovery flow.


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

### Upkeep States

The upkeeps states are used to track the status of log upkeeps (ineligible, performed) across the system, to avoid redundant work by the recoverer. Enables to select by (upkeepID, logIdentifier) is used as a key to store the state of a log upkeep.

The state is updated by the coordinator when the upkeep is performed or after ineligible check by observer.

The states will be persisted to so the latest state to be restored when the node starts up.

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
