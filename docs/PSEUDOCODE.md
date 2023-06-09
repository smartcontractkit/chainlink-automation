# Pseudocode Walkthrough of OCR3 Plugin Functionality
Because of the heavy modular approach, each module has its own pseudocode where
the detail of that module is discussed. Finally, an example putting some modules
together for a specific flow is provided at the end.

## Simple Ticker Data and Observer Call
A ticker is a very simple construct and is the primary driver of running the 
eligibility check flow. These tickers can be time tickers or anything else that
follows the same tick structure. They may be on timed intervals or they may only
trigger on specific conditions. Those conditions are contained within each
ticker.
```
for each tick
    call observer with tick data
```

## General Eligibility Check Flow (Observer)
The simplified modular flow uses data from an arbitrary tick to get upkeep
payloads from a registry. The upkeeps should be scoped to the tick type and any
data contained in the tick. This data could include upkeep payloads as well as
any other data.

Preprocessors act as filters, adders, or modifiers to upkeep payloads and return
a slice of upkeep payloads. 

The check pipeline runner contains cache data for check results (to reduce RPC
load) as well as check pipeline paralellization.

Postprocessing allows custom output hooks to be added to observers such that
check pipeline results can be delivered to vairous destinations.
```
get upkeep payloads for tick type

for each preprocessor
    call preprocessor with upkeep payloads

call check pipeline runner with preprocessed upkeep payloads
call postprocessor with results from check pipeline
```

### Log Trigger Preprocessing
When an upkeep check result is included in an outcome, the check result should
not be added to future outcomes. This preprocessing filters out check results
that have already been processed.

### Log Trigger Postprocessing
Log trigger postprocessing is a combination of putting eligible results into
OCR staging and sending retryable results to the retry ticker. The last part is
unique to log triggers and is not applicable to conditionals.
```
for each check pipeline result
    if check pipeline result is eligible
        add result to ocr staging
    else if check pipeline result is retryable
        add check result to retry ticker
```

### Conditional Preprocessing
Conditional upkeeps need to be continuously processed at progressing blocks. A
single upkeep can either be in-flight or transmitted as a stale report. OCR
reporting blocks upkeeps and chain events unblock them each ensuring that the
next perform will happen at on a block in the future.
```
for each upkeep payload
    if upkeep is pending or stale
        remove upkeep payload from provided list

return upkeep payload list
```

### Conditional Postprocessing
Contitional upkeeps require previous results that have not been processed to be
replaced by newly created results to ensure block numbers and results are kept
fresh.
```
remove results from previous tick from ocr staging

for each check pipeline result
    if check pipeline result is eligible
        add result to ocr staging
```

### Retry Postprocessing
Retries happen when errors are surfaced through the check pipeline. An upkeep
needs to be sent back through the check pipeline at a future time based on some
amount of timeout. Consider exponential backoff per upkeep as an example delay
mechanism.

A retry component can be structured as a combination of a ticker and 
postprocessor hook. Postprocessing provides data to the ticker and the ticker
runs an observer that pushes values into OCR staging.
```
[postprocessor]
for each check result
    if check result is retryable
        add to retry ticker

[retry ticks]
for time interval
    for each retryable upkeep payload
        if upkeep is retryable now
            add to next retries

send retries as tick
```

### Conditional Sampling
To reduce over all check load on the nodes for conditional upkeeps, each node
only checks a random subset of upkeeps per tick interval. The tick interval is
derived by block cadence, but not every chain will have every block checked.
Fast chains will have multiple blocks skipped while sampling takes place.
```

```

## OCR3 Report Flow
OCR is allowed to run on a separate cadence from the eligibility check flow. In
general, it pulls from some collection of staged values, constructs observations,
and builds reports. Some components hook into this process to feed progress
data to the eligibility check flows.

For the case of node restarts, it was proposed that local caches would be
cleared exposing a possibility that upkeeps could be reported by a set of 
restarted nodes resulting in successful (bad) transmits.

Approach #1: maintain recent results in outcome and use as a filter for new
outcomes. (not shown)

Approach #2: consider the lack of reporting an upkeep as a valid observation.
f+1 nodes can indicate that an upkeep needs to be performed while f+1 nodes can
also indicate that an upkeep does not need to be performed. In this case, both
observations would achieve a valid quorum, but produce different outcomes. 
Comparing the observations against each other, accounting for both the positive 
*AND* negative, a final unified outcome can be achieved that is resistant to
node restarts or fresh nodes entering the network. (not shown)

Approach #3: f+1 'votes' required to perform an upkeep. this is the simplest
approach with the least unknowns and is the one we're going with. (shown)
```
for each round
    // Observation -------------------
    for each outcome hook
        call hook run with previous outcome

    // filtering can be applied within ocr staging such that the previous add
    // to in-flight is immediately applied to results being pulled
    for each check pipeline result from ocr staging
        call peek on ocr staging queue to get result // does not remove
        add check pipeline result to observation

    get latest block from sample staging
    add latest block to observation

    // only eligible upkeeps
    get sampled upkeep ids from sample staging

    for each sampled upkeep id
        add upkeep id to observation

    // observation contains: latest block, sampled ids, check pipeline results
    return observation

    // Outcome -------------------
    for each attributedObservation
        increment count for check pipeline result
        update median block number with observation latest
        add observation id samples to sampled id list

    for each flattened check pipeline result
        if count is greater than f+1
            add to outcome

    flatten and dedupe sampled ids
    add sampled ids to outcome
    add median block to outcome

    return outcome

    // Report -------------------
    for each check pipeline result in outcome
        if next result does not push report over gas limit
            add result to report
        else
            start new report

    return array of reports
```

### Outcome Hooks
Multiple processes need to be run on the outcomes from the OCR observation. To
simplify and modularize this, the OCR process will run multiple hooks passing
the previous round's outcome to each hook.

### Coordinator Hook
```
get check pipeline results from outcome

for each check pipeline result
    call accept on coordinator with check pipeline result
```

### Check Pipeline Result Staging Hook
To allow results to be added to outcomes over multiple rounds, a result is 
pulled from the ocr staging queue using `peek` instead of `pop`. If a result is
in an outcome, the result should be popped from the queue so that it doesn't get
added to the next observation.
```
get check pipeline results from outcome

for each check pipeline result
    call pop on ocr staging queue for check pipeline result
```

### Create Coordinated Tick Hook
The coordinated ticker runs the check pipeline, producing results for 
coordinated candidates. 
```
get median block from outcome
get sample ids from outcome

call run on coordinated ticker with median block and sample ids
```