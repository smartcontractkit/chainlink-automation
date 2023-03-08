# What is an Encoder?
The responsibility of an encoder is to interpret raw `PointState` values and
provide specific operations that cannot be done purely on the raw values. One
example would be validation of `PointState` raw values.

An `Encoder` should ideally not contain any state and should simply be an
interpreter. All functions should be pure.

An `Encoder` can be used by an `Observer`, but in that case the `Observer`
and the `Plugin` should share the same `Encoder`. An `Encoder` typically maps to
a single contract instance or registry.

## Encoder
An `Encoder` is responsible for encoding/decoding upkeeps, observation points, 
performable results, etc. since each of these concepts are simply wrappers for
byte arrays. The `Plugin` should never interact directly with these values
without using the `Encoder`. The encoder can provide interface types only when
comparisons are needed.

## Interface
```
type Encoder interface {
    // Valid should validate that the individual raw values are correct in 
    // in structure. Error should be provided to bubble up specific errors
    // encountered in validation.
    Valid(PointState) (bool, error)
    // EncodeReport should pack as many PointState values into a single report
    // as possible (by configuration) and fully encode the result. The report
    // is registry specific so the plugin should have no knowledge of what is
    // inside.
    // An EncoderConfig can be included to pass config values from the off-chain
    // config to the encoder. This provides a way for network-wide configurations
    // to be used by an `Encoder`.
    EncodeReport([]PointState, EncoderConfig) ([]byte, error)
    // ExtractPointIdentifiers should return a list of identifiers for instances
    // of PointState included in a report. A PointIdentifier should include
    // what and when a PointState references such that an Observer can use this
    // value to mark an upkeep as in-flight and always progressing.
    ExtractPointIdentifiers([]byte) ([]PointIdentifier, error)
}
```