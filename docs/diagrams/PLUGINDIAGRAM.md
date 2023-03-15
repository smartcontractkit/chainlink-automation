# Automation LibOCR Plugin Flow Diagram

```mermaid
sequenceDiagram
    participant LibOCR
    participant Plugin
    participant Observer
    participant Coordinator
    participant Encoder

    par libOCR round sequence
        LibOCR->>Plugin: Query
        Note over LibOCR,Plugin: query is not required by automation
        Plugin-->>LibOCR: [empty bytes]
        LibOCR->>Plugin: Observation(Query)
        Plugin->>Observer: Observe(Config)

        Observer->>Observer: getLatestObserved

        loop each upkeep
            Observer->>Coordinator: IsPending()
            Coordinator-->>Observer: bool
            
            Observer->>Observer: addToObservation

            Observer->>Encoder: Eligible()
            Encoder-->>Observer: bool
        end

        loop each upkeep observed
            Observer->>Encoder: EncodeUpkeepIdentifier(upkeep)
            Encoder-->>Observer: UpkeepIdentifier
        end
        
        Observer->>Observer: getLatestBlock

        Observer-->>Plugin: (BlockKey, []UpkeepIdentifier, error)
        Plugin->>Plugin: shuffleAndLimit
        Plugin->>Plugin: makeVersionedObservation

        Plugin-->>LibOCR: [observation bytes from VersionedObservation]
        
        LibOCR->>Plugin: Report([]Observation)

        loop each observation
            Plugin->>Encoder: ValidateBlockKey()
            Plugin->>Plugin: addBlockKeyToList

            loop each UpkeepIdentifier
                Plugin->>Encoder: ValidateUpkeepIdentifier()
                Plugin->>Plugin: addIdentifierToList
            end
        end

        Plugin->>Plugin: findBlockKeyMedian

        loop each UpkeepIdentifier
            Plugin->>Encoder: MakeUpkeepKey(median, id)
            Encoder-->>Plugin: UpkeepKey

            Plugin->>Coordinator: IsPending()
            Coordinator-->>Plugin: bool

            Note over Plugin: only add to list if not pending
            Plugin->>Plugin: addToUpkeepKeyList
        end

        Plugin->>Plugin: shuffleAndDedupe

        Plugin->>Observer: MakeReport([]UpkeepKey)
        Observer->>Encoder: EncodeReport([]UpkeepKey)
        Encoder-->>Observer: [result]
        Observer-->>Plugin: []byte, error

        Plugin-->>LibOCR: [report bytes]
    and LibOCR transmit sequence
        LibOCR->>Plugin: ShouldAcceptFinalizedReport(Report)
        
        Plugin->>Encoder: KeysFromReport([]byte)
        Encoder-->>Plugin: ([]UpkeepKey, error)

        Plugin->>Coordinator: Accept([]UpkeepKey)
        Coordinator-->>Plugin: error

        Plugin-->>LibOCR: [bool]

        LibOCR->>Plugin: ShouldTransmitAcceptedReport(Report)
        alt contract v2.0
            loop each UpkeepKey
                Plugin->>Coordinator: IsTransmissionConfirmed(UpkeepKey)
                Coordinator-->>Plugin: [boolean]
            end
        else contract v2.02+
            Plugin->>Coordinator: IsTransmissionConfirmed(ReportTimestamp) (boolean)
            Coordinator-->>Plugin: [boolean]
        end
        Note over Plugin,Coordinator: v2.02 introduces report logs, v2.0 relies on individual upkeep logs
        Plugin-->>LibOCR: [bool]
    end
```