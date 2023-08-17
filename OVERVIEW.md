# Pseudocode

The following attempts to capture the general logic of the Automation OCR2
implementation using pseudocode. It is divided first by the OCR2 interface
followed by descriptions of secondary services.

## How an Observation is Constructed

An automation OCR2 observation is constructed of a block number and a shuffled
list of upkeep IDs (big.Int), json encoded to a limit of max bytes. The list
should only include eligible upkeeps (covered in a later section). Shuffling
is done using a randomness source derived from the config digest, epoch, and
block number. This ensures all nodes have the same source, but the results are
random across different OCR rounds if an observation is repeated.

> shuffling is done to ensure that all upkeeps have a chance at being sent in an
> observation due to limiting the observation length

```
ocr -> PluginThatImplementsLibOCR2
poller -> ParallelServicePollingLogs
filter -> ParallelServiceWatchingUpkeepLogs

call ocr.Observation with [ReportTimestamp]
    call poller.SampleUpkeeps with filter.Filter(): return [blockNumber] and [upkeepStateResults]

    define [upkeepIDList]
    for each (upkeepStateResults)
        if (upkeepState) is Eligible
            add (upkeepID) to (upkeepIDList)

    call shuffle with (upkeepIDList) and [randomSourceFromReportTimestamp]

    define [observation]
    add (blockNumber) to (observation)

    define [observationBytes]
    for each (upkeepIDList)
        add (upkeepID) to (observation)
        call encode with (observation): return [bytes]
            if length of (bytes) < [maxObservationLength]
                set (observationBytes) to (bytes)
                continue
            else
                remove (upkeepID) from (observation)
                call encode with (observation): return [bytes]
                set (observationBytes) to (bytes)
                break

    return (observationBytes)
```

## How a Report is Constructed

Observations from all other nodes (including the one processing a report) are
processed into a final encoded report ready to be transmitted on chain. Key
checks include:

- all observations that don't pass decoding are discarded
- a report must be built off of at least 1 observation
- all observations must have a block number to calculate median from
- observation id length should be less than or equal to max
- check keys (block+ID) should be less than max keys for a report
- all upkeeps should be checked at specified median block for eligibility
- report capacity is based on max upkeeps per report (currently configured as 1), though there exists report gas limit checks too

All upkeep ids are combined into a single list, duplicates removed, in-flight
upkeeps removed, and finally shuffled using the report timestamp such that a
shuffled result is random between rounds, but identical across all nodes for the
same list of upkeeps in the same round.

All upkeep ids are checked at the median block number by doing RPC calls to
chain state to verify eligibility.

Finally all eligible upkeeps are packaged into a report with consideration to
the gas required per upkeep, the maximum gas allowed per report, and the
configured maximum number of upkeeps allowed in a report.

### Median Block Number

All observations report a block number they observed upkeeps at along with the
upkeeps observed, if any. Since there should always be a block number in every
observation, the median is selected for constructing upkeep keys. The way the
median is selected is not a true median AND finally a configurable lag is
subtracted from the resulting median that puts the final checks behind the head
block even further.

If there are an odd number of observations, the middle value is selected just
like a true median. If there are an even number of observations, the highest of
the two middle values is selected instead of averaging them. Finally, the block
lag is subtracted from the selected median, resulting is what is being called
the `medianBlockNumber`.

```
ocr -> PluginThatImplementsLibOCR2
filter -> ParallelServiceWatchingUpkeepLogs

call ocr.Report with [ReportTimestamp] and [observationList]
    if length (observationList) == 0
        return false, nil report, and error

    define [allBlockNumbers]
    define [allUpkeepIDs]
    define [errorCount]
    for each (observationList)
        call decode [observation]: return [blockNumber], [upkeepIDList], and [error]
            if (error)
                add (1) to (errorCount)
                continue
            else
                add (blockNumber) to (allBlockNumbers)
                add (slice upkeepIDList:maxLengthPerObservation) to (allUpkeepIDs)
                continue

    if (errorCount) == length of (observationList)
        return false, nil report, and error

    call getMedian with (allBlockNumbers): return [medianBlockNumber]

    define [upkeepKeys]
    for each (allUpkeepIDs)
        call combine with (medianBlockNumber) and (upkeepID): return [upkeepKey]
            add (upkeepKey) to (upkeepKeys)

    call dedupe with (upkeepKeys)

    for each (upkeepKeys)
        call filter.Filter() with [upkeepKey]: return [isPending]
            if (isPending)
                remove (upkeepKey) from (upkeepKeys)

    call shuffle with (upkeepKeys) and [randomSourceFromReportTimestamp]

    if length of (upkeepKeys) > [maxReportKeysLimit]
        set (upkeepKeys) to (slice upkeepKeys:maxReportKeysLimit)

    if length of (upkeepKeys) == 0
        return false, nil report, and no error

    define [toPerform]
    for each (upkeepKeys)
        call CheckUpkeep with [upkeepKey]: return [upkeepStateResult]
            if [upkeepState] is not Eligible
                continue

            if [reportCapacity]+[upkeepGasUsed] > [maxReportGas]
                continue

            add (upkeepStateResult) to (toPerform)

            if length of (toPerform) >= [maxUpkeepsInReport]
                break

    if length of (toPerform) == 0
        return false, nil report, and no error

    call encode with (toPerform): return [bytes]

    return true, (bytes), and no error
```

## How a Report is Accepted

In the acceptance of a report, the details of the incoming report must match the
following:

- byte length should be greater than zero
- at least one upkeep in report
- no decode error
- no errors from marking a key as 'in-flight' (which would only happen for an error decoding an upkeep key)
- otherwise, all keys are accepted

This step is primarily focused on minimal report verification and extracting
values from the report for tracking internal state. This internal state is used
to provide more accurate observations and verify observations in report
generation.

```
ocr -> PluginThatImplementsLibOCR2
filter -> ParallelServiceWatchingUpkeepLogs

call ocr.ShouldAcceptFinalizedReport with [ReportTimestamp] and [ReportBytes]
    if length of (ReportBytes) == 0
        return false, and no error

    call decode with (ReportBytes): return [upkeepStateResultList]
    if err in decode
        return false, and decode error

    if length of (upkeepStateResultList) == 0
        return false, and error

    for each (upkeepStateResultList)
        call filter.Accept with [upkeepStateResult]
        if err in accept
            return false, and error

    return true, and no error
```

## How a Report is Transmitted

Since multiple nodes will attempt to transmit a report in sequence, the decision
on whether to transmit is based on the following:

- no decode error of report
- at least one upkeep in report
- at least one upkeep from report not confirmed

Confirmation of upkeeps is based on logs emitted from the contract and more
detail is provided in the `filter` description.

```
ocr -> PluginThatImplementsLibOCR2
filter -> ParallelServiceWatchingUpkeepLogs

call ocr.ShouldTransmitAcceptedReport with [ReportTimestamp] and [ReportBytes]
    call decode with (ReportBytes): return [upkeepStateResultList]
    if err in decode
        return false, and error

    if length of (upkeepStateResultList) == 0
        return false, and error

    for each (upkeepStateResultList)
        call filter.IsTransmissionConfirmed with [upkeepStateResult]
        if upkeep is not confirmed
            return true, and no error

    return false, and no error
```

## Parallel Services

### Poller

This service interfaces with the registry contract and pulls a list of active
upkeeps for every block. It is up to the registry to maintain the list of
active upkeeps either by itself polling the contract or by listening for logs.

**Sampling Ratio**

The sampling ratio is defined as the ratio of upkeeps a single node MUST check
at random such that `n` good nodes have `p` probability of checking all upkeeps
at least once over `r` range of blocks.

This limits the workload of individual nodes and attempts to distribute the
total workload equally to all nodes while synchronizing only on `n`, `p`,
and `r`.

Faulty node limit `f` is included in the calculation of the sampling ratio.

**Polling**

For each block, this service attempts to check the state of all active upkeeps
provided by the registry. First, all upkeeps are shuffled before checking
their state. This ensures that all upkeeps have a chance at being checked over a
series of blocks. Second, the list of active upkeeps is reduced to the sampling
size limit.

Each upkeep is then checked at the provided block number and the total results
are used to reset an internal cache such that the results for the latest block
replace the results for the previous block.

```
contract -> RegistryContractInterface

for each [Block]
    define [activeUpkeeps]
    for each [upkeep]
        if upkeep is active and not cancelled
            add (upkeep) to (activeUpkeeps)

    call shuffle with (activeUpkeeps)
    replace (activeUpkeeps) with slice of (activeUpkeeps:sampleRatioLimit)

    define [upkeepResults]
    for each (activeUpkeeps)
        call contract.CheckUpkeep with [upkeep] and (Block): return [upkeepResult]
        if no error
            add (upkeepResult) to (upkeepResults)

    replace (poller.upkeepResultsCache) with (upkeepResults)
```

**Providing Samples**

Samples are provided directly from the cached results of the last poll. All
'in-flight' results are filtered out at this point.

```
if no results sampled
    return no results, and error

define [blockNumber]
define [upkeepResults]
for each (upkeepResults)
    if [upkeepResult] is Pending
        remove (upkeepResult) from (upkeepResults)

return (blockNumber), (upkeepResults), and no error
```

### Filter

The filter is a coordinator between observations, building reports, and
transmitting reports. It runs as a service, checking logs emitted from the
contract to evaluate the pending state of each upkeep.

**Accepting as In-flight**

This is the set point for the internal state. Upkeeps that are accepted pass
through this logic to update upkeep state. There are two states necessary to
ensure an upkeep doesn't encounter a double-transmit scenario.

- block and id for transmit confirmation
- id only to filter out of observations

In this logic, an `upkeepKey` is the composition of both the block number and
the unique upkeep identifier.

- accepting a key should only return an error if the key is not parsable
- accepting a key should always block the `upkeepKey`
- if the key is already set, the id should be blocked for the highest block number between the previously blocked or the block number from the `upkeepKey`

```
define [upkeepKey]
define [pendingKeys]
define [blockedIDs]

if (upkeepKey) is not in (pendingKeys)
    set (upkeepKey) in (pendingKeys) with expiration of [pendingKeyTimeout] and confirmed false

    split (upkeepKey) into [blockNumber], [upkeepID]

    if (upkeepID) is not in (blockedIDs)
        set (upkeepID) in (blockedIDs) with (blockNumber) and [maxTransmitBlockNumber]
    else
        define [blockedIDBlockNumber]

        if (blockedIDBlockNumber) is after (blockNumber)
            set (upkeepID) in (blockedIDs) with (blockNumber) and [maxTransmitBlockNumber]
```

**Filtering**

Once an `upkeepKey` and `upkeepID` are blocked, the upkeep will be filtered out
of results until one of two things occur:

- the blocked key and id encounter the `pendingKeyTimeout`
- a perform log is detected for the upkeep id (conditional to transmit block)

Any newly accepted `upkeepKey` will have a transmit number set to the max
allowable value. In this case, the block number for the `upkeepKey` will always
be before the last transmit block number resulting in filtering out the upkeep
from observations.

When an `upkeepKey` is compared against a recently performed upkeep, the
transmit block number will be the one indicated in the perform log and the
block number from the `upkeepKey` will be compared against the perform log's
transmit number. If an observed `upkeepKey` occured at a block previous to the
last transmit block number, the key will be filtered out of the observation.
This ensure observations are not stale based on tracked internal state.

```
define [upkeepKey]
define [blockedIDs]
define [lastTransmitBlock]

split (upkeepKey) into [blockNumber], [upkeepID]

if (upkeepID) is in (blockedIDs) and (blockNumber) is before or equal to (lastTransmitBlock)
    filter out (upkeepKey)
```

**Transmitting**

Transmission confirmation is done on the `upkeepKey` directly. If there is a key
in pending state that matches the `upkeepKey` exactly, the perform action has
not yet been confirmed.

```
if (upkeepKey) is not in (pendingKeys)
    return transmissionConfirmed
else if (upkeepKey) is in (pendingKeys) and (confirmationState) is true
    return transmissionConfirmed
else
    return transmissionNotConfirmed
```

**Logs**

Logs are polled every second and passed through the following:

Perform log entries indicate that a perform exists on chain in some capacity.
The existance of an entry means that the transaction was broadcast by at least
one node. Reorgs can still happen causing performs to get moved to a later block
or change to reorg logs. Higher minConfirmations setting reduces the chances of
this happening.

We do two things upon receiving a perform log:

- Mark the upkeep key responsible for the perform as 'transmitted', so that this node does not waste gas trying to transmit the same report again
- Unblock the upkeep from idBlocks so that it can be observed and reported on again.

```
define [performLogs]
define [configMinConfirmations]
define [pendingKeys]
define [blockedIDs]

for each (performLogs)
    define [upkeepKey]
    define [performLogConfirmations]
    define [performBlockNumber]

    if (performLogConfirmations) >= (configMinConfirmations)
        split (upkeepKey) into [blockNumber], [upkeepID]

        if (upkeepKey) in (pendingKeys)
            define [upkeepKeyConfirmationState]

            if (upkeepKeyConfirmationState) is false
                set (upkeepKey) in (pendingKeys) with expiration of [pendingKeyTimeout] and confirmed true

                set (upkeepID) in (blockedIDs) with (blockNumber) and (performBlockNumber)
            else
                if (upkeepID) is in (blockedIDs) and
                    (blockNumber) == [blockedIDBlockNumber] and
                    (performBlockNumber) != [blockedIDBlockNumber]

                    set (upkeepID) in (blockedIDs) with (blockNumber) and (performBlockNumber)

```

It can happen that in between the time the report is generated and it gets
confirmed on chain something changes and it becomes stale. Current scenarios are:

- Another report for the upkeep is transmitted making this report stale
- Reorg happens which changes the checkBlockHash making this reorged report as it was checked on a different chain
- There's a massive gas spike and upkeep does not have sufficient funds when report gets on chain

In such cases the upkeep is not performed and the contract emits a log
indicating the staleness reason instead of UpkeepPerformed log. We don't have
different behaviours for different staleness reasons and just want to unlock the
upkeep when we receive such log.

For these logs we do not have the exact key which generated this log. Hence we
are not able to mark the key responsible as transmitted which will result in
some wasted gas if this node tries to transmit it again, however we prioritize
the upkeep performance and clear the idBlocks for this upkeep.

```
define [performLogs]
define [configMinConfirmations]
define [pendingKeys]
define [blockedIDs]

for each (performLogs)
    define [upkeepKey]
    define [performLogConfirmations]
    define [performBlockNumber]

    if (performLogConfirmations) >= (configMinConfirmations)
        split (upkeepKey) into [blockNumber], [upkeepID]

        define [nextBlock]
        set (nextBlock) to (blockNumber) + 1

        if (upkeepKey) in (pendingKeys)
            define [upkeepKeyConfirmationState]

            if (upkeepKeyConfirmationState) is false
                set (upkeepKey) in (pendingKeys) with expiration of [pendingKeyTimeout] and confirmed true

                set (upkeepID) in (blockedIDs) with (blockNumber) and (nextBlock)
            else
                if (upkeepID) is in (blockedIDs) and
                    (blockNumber) == [blockedIDBlockNumber] and
                    (performBlockNumber) != [blockedIDBlockNumber]

                    set (upkeepID) in (blockedIDs) with (blockNumber) and (nextBlock)

```
