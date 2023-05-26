# Generic Ticker Sequence

A generic sequence applies middleware to pre-process upkeep payloads either to 
filter or modify upkeep payloads before running the check pipeline on each.
Finally, a single post-processing is applied with the array of results from the
check pipeline.

The concept of retries can also be done by routing results from the
post-processor back to the pre-processing middleware and inject them back into
the check process.

```mermaid
sequenceDiagram
    participant TK as Ticker[T any]
    participant OB as Observer[T any]
    participant R as Registry
    participant MD as PreProcessor
    participant RN as Runner
    participant P as PostProcessor
    
    TK->>OB: Process(ctx, T)
    OB->>R: GetActiveUpkeeps(T)
    R-->>OB: []UpkeepPayload, error

    Note over OB,R: NOTE: GetActiveUpkeeps only returns upkeeps for type T

    loop for each PreProcessMiddleware
        OB->>MD: Run([]UpkeepPayload)
        MD-->>OB: []UpkeepPayload, error
    end

    OB->>RN: CheckUpkeeps(ctx, []UpkeepPayload)
    RN->>RN: makeBatches()
    loop for each batch execute concurrently
        RN->>R: CheckUpkeeps(ctx, []UpkeepPayload)
        R-->>RN: []CheckResult, error
    end
    RN-->>OB: []CheckResult, error

    OB->>P: Run([]CheckResult)
    P-->>OB: error
```