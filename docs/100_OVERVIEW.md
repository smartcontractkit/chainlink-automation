# OCR2 Automation Design Overview
The intent of the plugin design is to separate responsibility of components,
ensure modularity, and provide accessible types for viewing content in 
messages via external sources (like telemetry).

## Glossary of Terms
- Observation: two different types; one from the `Plugin` and one from an `Observer`
- Report: encoded instructions and/or values for on-chain execution
- BlockKey: a block identifier (typically a big.Int on many chains)
- UpkeepIdentifier: an identifier for an upkeep (typically a big.Int)
- UpkeepKey: a combination of `BlockKey` and `UpkeepIdentifier`
- NetworkConfig: also called off-chain config in OCR; network configuration
- In-flight: upkeeps are in-flight if they are reported, but not transmitted
- Transmit: an attempt to include a report on-chain

## Components

### Plugin
The `Plugin` provides methods for LibOCR to call as a valid OCR2 plugin. The 
responsibility of each method is defined below. Generally, a `Plugin` gets
observations from a list of multiple `Observer`, uses a `Merger` to merge
observations from the network, and an `Encoder` to produce a transmittable
report.

#### Rules
- A `Plugin` only interacts with `Observer`, `Merger`, and `Encoder`
- A `Plugin` should not know anything about the structural contents of a `BlockKey`
- A `Plugin` should not know anything about the structural contents of a `UpkeepIdentifier`
- A `Plugin` should start and stop services per its appropriate life cycle
- A `Plugin` should treat all incoming observations as potentially malicious
- A `Plugin` should isolate all state interactions to instances of `Observer`

#### Observation
The `Observation` method should loop through multiple `Observer` instances to
collect pending and eligible upkeep results. Those results should be packaged 
together as an automation observation (distinct type) and byte encoded. This 
function should never directly interact with the encoded upkeep results. Its 
sole responsibilities should remain as a collector and basic encoder to the 
final transport type.

More about specifics of an observation from an observer here:
[OBSERVATION](./300_OBSERVATION.md)

##### Observation Struct
An observation is composed of a list of eligible upkeeps and a list of pending
upkeeps. Providing both allows for all nodes to produce a report without
checking internal state. Versioning allows for version checks to be applied as
new nodes can join the network with different active versions.

```
type VersionedObservation struct {
    // Version is intended to allow for observations to change over time
    Version     string
    // Encoding indicates the type of encoder/registry used to build the
    // included observations
    Encoding    string
}

// ObservationV10 is the intended first step in observation versioning
type ObservationV10 struct {
    VersionedObservation
    // Block contains the block value for nodes to coordinate on
    Block BlockKey
    // Data contains actual values of this version of observation
    Data []UpkeepIdentifier
}

// ObservationV11 displays a potential way to differentiate observation versions
type ObservationV11 struct {
    VersionedObservation
    // Block contains the block value for nodes to coordinate on
    Block BlockKey
    // Data contains actual values of this version of observation
    Data       []UpkeepIdentifier
    // OtherStuff in this case is addative and the version minor is incremented
    OtherStuff int
}

// ObservationV20 displays a potential way to differentiate observation versions
type ObservationV20 struct {
    VersionedObservation
    // Data in this version is a different structure and would break 
    // compatibility to previous versions
    Data       map[int]PointState
}
```

#### Report
A report should be able to be constructed without ever checking internal or
external state. All observations should be fully validated independently, merged 
together, and reduced by provided pending state structs. The result should be 
entirely stateless.

More information on building reports here: [REPORT](./400_REPORT.md)

#### Life Cycle
An instance of a `Plugin` is produced by a factory method. This factory method
should inject all necessary dependencies and start all necessary background
processes. In the context of LibOCR, an instance of a `Plugin` only lives as
long as the latest config digest. If a new config digest is detected, a new
instance of a `Plugin` is created and the old instance is stopped. This being
the case, a `Plugin` instance should fully respect a call to stop required
background processes and end all internal activities gracefully.

### Observer
An `Observer` evaluates upkeep state from a chain registry and collects upkeeps
that need to be performed along with data required for comparison and reporting.

Examples can include:
- polling observer (polls the chain at each block)
- log trigger observer (logs can trigger observations)
- scheduled observer (polling occurs on a defined schedule like cron)

More information on observers here: [OBSERVER](./350_OBSERVER.md)

### Encoder
An `Encoder` is intended to be purely a translation layer. A `Plugin` should not
need to know the structural contents of observations or reports, which is why an 
`BlockKey` and `UpkeepIdentifier` contain raw byte values. The responsibility 
of an `Encoder` is to convert those raw values into necessary actionable results.

More information on encoders here: [ENCODER](./500_ENCODER.md)

## External Message Interactions
The use of OCR telemetry was considered in the design of the plugin such that
external observers could have the means of extracting specific values of 
observations while maintaining strict isolation of responsibilities.

The intended approach for extracting data from observations involves decoding
the concrete plugin observation type, `VersionedObservation`, and further
extracting values from the underlying observation using the appropriate encoder.

This allows concrete types to be used directly by the plugin, where observation
comparison is concerned, while leaving encoding a separate and modular 
responsibility.

Observation distinct types and decoders should be exposed as public types and
functions for the package.

More information on the external package interface and types: [TYPES](./700_TYPES.md)

## Open Questions
- Does LibOCR leverage compression on message delivery?
- Would it be helpful to implement data compression on observations or reports?
- What should be the observation size limits? (check data, perform data, etc.)
- What should be report size limits? (perform data is a big factor)
- How is the best way to pass off-chain config values to plugin components?
