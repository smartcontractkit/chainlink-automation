# Generic Observer
An `Observer` evaluates upkeep state from a chain registry and collects upkeeps
that need to be performed along with data required for comparison and reporting.

Examples can include:
- polling observer (polls the chain at each block)
- event trigger observer (logs can trigger observations)
- scheduled observer (polling occurs on a defined schedule like cron)

## Observer Rules
- an observer should be chain agnostic
- an observer should interact with a registry through a wrapper
- an observer can define its own dependency interfaces
- an observer should track relevant upkeep pending state
- an observer can have registry specific structures
- all observers used by the same plugin instance should use the same registry
- an observer should provide an active window (time frame) of observations

## Active Window
In the context of a block chain, an active window is a block frame between block
`x` and block `y`. An observer should provide observations within a defined or 
configured active window even if the same `upkeep` is repeated every block
within that active window. An observation should go stale once the chain
advances outside the active frame and should not be included in the filtered
output.

The active window provides enough observations to other nodes such that common
overlaps that satisfy the quorum requirements can be deterministically acheived
by multiple nodes.

## Size of an Observation

Since an OCR observation in the context of automation is a list of `PointState`
within an active window, the size of an observation can grow significantly.
However, an upper limit can be calculated based on the `PointState` value sizes
and the limitations on calldata per chain.

**Assumptions**
- the report gas limit is 5 million
- a single upkeep gas limit is allowed to equal the report limit (5 million)
- the evm estimates 16 gas per non-zero byte of data
- an identifier is a uint64 value (8 bytes)
- an active window is 12s
- throughput is 3 upkeeps per second average
- each node has a 1/12 chance of being leader (leader is required to broadcast)

With the perform data of an upkeep being much larger than all other values in
a `PointState`, we can discard the `Identifier`, `Hash`, and `Ordinal` as 
insignificant values on estimating the size of an observation. In that case,
we work with the assumptions above.

```
approx. # of bytes = (5,000,000 / 16) * 3 * 12 = 11,250,000 = 11.4 MB
```

This calculation assumes an upward bound of 15 million gas used per second to
maintain average throughput. Uncompressed, this is about the size of a RAW image
file. If gzip compression is used for message communication, this value can be 
reduced. Uncompressed, one observation would take 1 second to upload on a
100 Mbps connection, assuming a node is running on a high-speed home internet
connection (which shouldn't be the case).

## Filter
A `Filter` tracks the pending state of upkeeps from the perspective of a single
node. Once an upkeep is accepted into a report, the filter should indicate
that upkeep as pending until a completion log is encountered or after a lockout
window. A `Filter` should be directly paired with an `Observer` as they work in
tandem to ensure accurate observations. The `Plugin` should never interact 
directly with a `Filter`.

## Interface
An `Observer` provides two functions as described in the following interface.

```
type Observer interface {
    Observe() ([]PointState, error)
    Accept([]PointIdentifier) error
}
```

# Polling Observer
The polling observer is a specific implementation of an `Observer` that polls a
chain through a registry every block. To reduce workload, the polling observer
limits the number of upkeeps check based on a sampling ratio.

[DIAGRAM](./diagrams/POLLINGOBSERVER.md)

## Ratio
The sampling ratio is determined by the probability that all upkeeps will be
observed by at least `n` nodes over `x` number of blocks.
