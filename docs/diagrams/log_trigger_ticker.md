# Log Trigger

A ticker and observer are paired on ticker data type. The registry provides
upkeep data and the check pipeline. The runner provides caching and 
parallelization and has the same interface as the check pipeline. The encoder
determines eligibility and finally eligible results are added to a queue.

On startup, an observer builds a mapping of log events to upkeeps and begins
watching the registry for upkeep configuration changes or new/cancelled upkeeps.
Upkeep changes not shown in the following diagram for simplicity.

```mermaid
sequenceDiagram
    participant TK as Ticker[T any]
    participant OB as Observer[T any]
    participant R as Registry
    %% TODO: participant Coordinator
    participant RN as Runner
    participant Q as Queue
    
    TK->>OB: Process(ctx, T)
    OB->>R: GetActiveUpkeeps(T)
    R-->>OB: []UpkeepPayload, error

    Note over OB,R: NOTE: GetActiveUpkeeps only returns upkeeps for type T

    OB->>RN: CheckUpkeeps(ctx, []UpkeepPayload)
    RN->>RN: makeBatches()
    loop for each batch execute concurrently
        RN->>R: CheckUpkeeps(ctx, []UpkeepPayload)
        R-->>RN: []CheckResult, error
    end
    RN-->>OB: []CheckResult, error

    loop for each result
        alt is eligible
            OB->>Q: Add(CheckResult)
        end
    end
```