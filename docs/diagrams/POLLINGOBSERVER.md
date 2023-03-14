# Polling Observer Sequence Diagram

> PollingObserver implements the `Observer` interface, but details of that
> interface are not shown here.
>
> A potential attack vector is flooding a registry with eligible upkeeps. As a
> mitigation, limit the number of eligible upkeeps per head and randomly select
> them from the total set of eligible before adding them to the cache.

```mermaid
sequenceDiagram
    participant Plugin
    participant HeadSubscriber
    participant PollingObserver
    participant Registry
    participant Encoder

    par per block sequence
        HeadSubscriber->>PollingObserver: NewHead()
        note over HeadSubscriber,PollingObserver: channel -> push new head

        PollingObserver->>Registry: GetAllActiveIDs()
        Registry-->>PollingObserver: []UpkeepID

        par
            note over PollingObserver,Registry: parallelize for each ID
            PollingObserver->>Registry: GetState(ID, CurrentBlock)
            Registry->>Encoder: EncodeAsObservation()
            Encoder-->>Registry: PointState
            Registry-->>PollingObserver: PointState
            PollingObserver->>PollingObserver: SetCache(Key, PointState)
        end
    and track pending state
        PollingObserver->>Registry: IsPending(PointIdentifier)
        Registry-->>PollingObserver: boolean
    and OCR observation sequence
        Plugin->>PollingObserver: Observe()
        PollingObserver->>PollingObserver: GetCacheValues()
        PollingObserver-->>Plugin: []PointState
    and OCR accept sequence
        Plugin->>PollingObserver: Accept([]PointIdentifier)
        PollingObserver-->>Plugin: boolean
    end
```