# Sampling Ticker

The sampling ticker checks a random sample of upkeeps and applies the ticker
and upkeep ids to a staging queue. The plugin pulls values from this queue to
acheive quorum on the results.

```mermaid
sequenceDiagram
    participant TK as Ticker[T any]
    participant OB as Observer[T any]
    participant R as Registry
    participant RN as Runner
    participant C as Coordinator
    participant S as SampleStager
    
    TK->>OB: Process(ctx, T)
    OB->>R: GetActiveUpkeeps(T)
    R-->>OB: []UpkeepPayload, error

    Note over OB,R: NOTE: GetActiveUpkeeps only returns upkeeps for type T

    OB->>OB: sampleSlicePayload()

    OB->>RN: CheckUpkeeps(ctx, []UpkeepPayload)
    RN->>RN: makeBatches()
    loop for each batch execute concurrently
        RN->>R: CheckUpkeeps(ctx, []UpkeepPayload)
        R-->>RN: []CheckResult, error
    end
    RN-->>OB: []CheckResult, error

    loop for each result
        OB->>C: IsPending(CheckResult)
        C-->>OB: bool, error

        alt is eligible and is not pending
            OB->>OB: addResult(UpkeepIdentifier)
        end
    end
    
    Note over OB,S: NOTE: sampled results are pushed to the stager
    OB->>S: Next(T, []UpkeepResult)
```