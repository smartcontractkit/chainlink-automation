# External Data Types and Interfaces

Some external services need concrete types to work with when encoding/decoding
raw messages between nodes. Other external services use concrete types or
interfaces to inject dependencies into the plugin.

## Factory Methods

All necessary factory methods for creating a new plugin instance should be
contained directly in `/pkg` and not be nested.

## Observers

Observers should be placed in a nested package `/pkg/observers` to make them
publicly available as injectable dependencies. All observers should adhere to
interfaces required in factory methods.

## Observation

An observation in the context of automation OCR as defined more in 
[OBSERVATION](./300_OBSERVATION.md) should be kept in `/pkg` to make them
directly accessible in an import.

## Off-chain Config

The off-chain config concrete type and associated encoding/decoding methods
should be contained in `/pkg` to be easily imported along with observation
concrete types.
