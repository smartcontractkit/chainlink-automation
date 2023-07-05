# Eligibility Flow

The eligibility flow is the sequence of events used to determine if an upkeep is
considered eligible to perform. This involves some input on a regular interval,
a series of pre-processing, checking via RPC calls, collecting data from HTTP
calls, and finally some type of post-processing. The output is expected to be
only upkeeps eligible to be performed on chain.

Required data includes a collection of active, configured upkeeps with an
unknown eligibility status.

[Generic Sequence Diagram](./diagrams/generic_ticker_sequence.md)

## Basic Value Types

To maintain chain agnostic behavior, the following data types should be
abstracted from the plugin as much as possible. That is, the less detail the
plugin needs from each of the following value types, the better.

Q. How are interface types used within the plugin? If the plugin doesn't care about them then why do they need to be passed to plugin at all?

### ConfiguredUpkeep

A configured upkeep represents all information that identifies a single upkeep
as registered by the user including upkeep id, trigger configuration, 
check data, etc.

```golang
type UpkeepIdentifier []byte

type ConfiguredUpkeep struct {
    // ID uniquely identifies the upkeep
    ID UpkeepIdentifier
    // Type is the event type required to initiate the upkeep
    Type int
    // Config is configuration data specific to the type
    Config interface{}
}
```

### UpkeepPayload

An upkeep payload represents all information the check pipeline requires to
check the eligibility of an upkeep and might include upkeep id, block number,
check data, etc.

This can potentially be a combination of both `ConfiguredUpkeep` and `T`:

```golang
type UpkeepPayload struct {
    // Upkeep is all information that identifies the upkeep
    Upkeep ConfiguredUpkeep
    // Tick is the event that triggered the upkeep to be checked
    Tick interface{}
}
```

### CheckResult

An upkeep check result represents all information returned from the check
pipeline for a single upkeep and might include eligibility, check block, upkeep
id, etc.

```golang
type CheckResult struct {
    // UpkeepPayload is the original payload request
    UpkeepPayload
    // Retryable indicates whether the result failed or not and should be retried
    Retryable bool
    // Eligible indicates whether the upkeep is performable or not
    Eligible bool
    // Data is a general container for raw results that pass through the plugin
    Data interface{}
}
```

## Pre-Processing and Post-Processing

Each interval type, *ticker*, initiates a payload collection, filtering, and 
checking. Pre-processing and post-processing provides unique behavior to 
different flow types where filtration, additions, or modifications to upkeeps is
required. Some examples follow.

### Upkeep In-Flight Status Coordination

Upkeeps have an *in-flight* status where an upkeep identified as eligible by a
quorum, but not yet transmitted on-chain must be filtered out of observations
sent to the OCR process. This is an example of pre-processing.

In combination with *in-flight* status, conditional upkeeps also have a lockout
mechanism to ensure that eligibility checking and performs continues to progress
in blocks. For example, if a conditional upkeep stale report log has been 
received for block 10, all further checks must be after block 10 and all checks
at or before block 10 should be filtered out. This is an example of 
post-processing, but might also be applied to pre-processing.

[Upkeep Eligibility Sampling Flow](./diagrams/sampling_ticker.md)

### Upkeep Check Retries

Some upkeeps are retryable in the event of check failures as indicated by the
check pipeline. Retry logic can be a connection between post-processing and 
pre-processing by identification and removal in post-processing, wait, and
re-injection in pre-processing.

### Upkeep Perform Queue

Different eligibility flows require different interactions with a perform queue
or even no interaction at all. A post-processor provides custom functions to
alter the eligibility flow as needed.

#### Sampling Flow

In the case of the sampling flow, the final destination of check results isn't
the perform queue. Instead it is a block and id list queue. This is a special
queue used by the automation OCR protocol to create coordinated ticks where
blocks and id lists are collected and agreed on by the network before applying
validation checks.

[Upkeep Eligibility Sampling Flow](./diagrams/sampling_ticker.md)

#### Coordinated Flow

In the case of the coordinated flow, the final destination is the perform queue
with the need to replace values from the previous check results interval with
the latest results.

[Coordinated Eligibility Flow](./diagrams/coordinated_ticker.md)

#### Log Trigger Flow

For log triggers, the final destination is also the perform queue without the
need to replace items in the queue. Log triggers are a single direction push
with the expectation that all values added to the queue will be performed.

[Log Trigger Eligibility Flow](./diagrams/log_trigger_ticker.md)