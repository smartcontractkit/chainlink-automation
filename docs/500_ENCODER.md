# What is an Encoder?
The responsibility of an encoder is to interpret raw `PointState` values and
provide specific operations that cannot be done purely on the raw values. One
example would be validation of `PointState` raw values.

An `Encoder` should ideally not contain any state and should simply be an
interpreter. All functions should be pure.

An `Encoder` can be used by an `Observer`, but in that case the `Observer`
and the `Plugin` should share the same `Encoder`. An `Encoder` typically maps to
a single contract instance or registry.

Ideally an encoder should not maintain any state and should be able to be 
represented with static functions instead of reference functions.

## Encoder
An `Encoder` is responsible for encoding/decoding upkeeps, observation points, 
performable results, etc. since each of these concepts are simply wrappers for
byte arrays. The `Plugin` should never interact directly with these values
without using the `Encoder`. The encoder can provide interface types only when
comparisons are needed.

## Interface

```
type Encoder interface {
    ValidateUpkeepKey(UpkeepKey) (bool, error)
    ValidateUpkeepIdentifier(UpkeepIdentifier) (bool, error)
    ValidateBlockKey(BlockKey) (bool, error)
    MakeUpkeepKey(BlockKey, UpkeepIdentifier) (UpkeepKey)
    SplitUpkeepKey(UpkeepKey) (BlockKey, UpkeepIdentifier, error)
    // EncodeReport should pack as many upkeep results into a single report
    // as possible (by configuration) and fully encode the result. The report
    // is registry specific so the plugin should have no knowledge of what is
    // inside.
    // A Config can be included to pass config values from the off-chain
    // config to the encoder. This provides a way for network-wide configurations
    // to be used by an `Encoder`.
    EncodeReport([]UpkeepResult, ...Config) ([]byte, error)
    // EncodeUpkeepIdentifier should take the output of a registry check result
    // as input and return an UpkeepIdentifier
    EncodeUpkeepIdentifier(UpkeepResult) (UpkeepIdentifier, error)
    // KeysFromReport extracts all upkeep keys from a report byte array
    KeysFromReport([]byte) ([]UpkeepKey, error)
    // Eligible takes an upkeep result to provide an abstraction per registry;
    // returns whether an upkeep is eligible or not along with any decoding
    // or type conversion errors encountered
    Eligible(UpkeepResult) (bool, error)
}
```