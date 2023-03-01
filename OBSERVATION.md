# What is an Observation?
An observation in the sense of Automation is the state of a single upkeep observed
at a single point in time (i.e. block number). Upkeep state is registry dependent, 
but generally consists of whether the upkeep should be performed, the data the
upkeep was checked with, the point at which the upkeep was checked, and the data
the upkeep should be performed with. Other information may also apply per
registry implementation. From here we will call this a `PointState`.

## Comparison
A `PointState` observed by one node can be directly compared with a `PointState`
observed by another node by simply comparing a hash of the data. If the hash is
equal, the two nodes observed the same state. This provides more of a deep equal
comparison, but because the observation timeline is linear it is also important
to compare whether one `PointState` is after or before another.

### Equal
Equality can only be determined do performing a deep equality check between two
instances of `PointState`.

### Congruent
Two instances of `PointState` could be said to be congruent if they both share
the same identifier. That is, the two instances reference the same upkeep. They
can be congruent AND equal. They can NOT be incongruent and equal.

### Ordinal
One instance of `PointState` can be congruent, but have occured after another
instance. Typically this ordinal is a block number, but doesn't strictly have to
be so.

## Structure
To provide the three functions necessary to build reports from multiple
observations, the structure should contain values that make each comparison 
operation as simple as a value comparison.

```
type PointState struct {
    Hash        [8]byte // hash of Data for deep equality check
    Identifier  [8]byte // provides congruence check
    TimeOrdinal [8]byte // allows points to be time ordered
    Data        []byte  // provides complete data required to build a report
}
```

## Merging Observations
Reports are created from the input of multiple observations from multiple nodes.
It's worth noting the distinction between an observation in the sense of OCR
and an observation as described above. The former is composed of multiple of the
latter.

In merging, the goal is to take multiple observations from outside sources and
distill them into a list of `PointState` that satisfies the following:

- initially: the occurance count of `PointState.Hash` across all observations is greater than configurable threshold
- second: if multiple occurances of `PointState.Identifier` are found that meet the occurance threshold, the latest as defined by `PointState.TimeOrdinal` is chosen
- finally: no duplicates of `PointState.Identifier`

The final result is a list of `PointState` that can be used to generate a report
for transmission.

## Other observation notes
In previous version of Automation, the contract enforced safety checks. If an
upkeep was cancelled in the contract, there was a grace period where the upkeep
could still be transmitted even though the contract registered it as cancelled.

In the new version of Automation, information like `cancelled` and
`perform gas limit` should be included in a `PointState` as this information
establishes whether one node the same upkeep state as another. Anything that can
influence the ability to perform the upkeep is part of that observed state.

This also includes the last point at which an upkeep was performed. The first
iteration of OCR Automation juggled logs to determine if an observed upkeep
could be performed or not, relating to the last perform. If this data is 
included in the `PointState`, perform safety assurances rest on the OCR quorum
instead of node state or contract controls.

For re-org security, the `PointState` should also include the block hash at
which the upkeep state was observed. This will also ensure that multiple nodes 
find agreement on complete blockchain state.