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
- an observer should provide block and upkeepidentifiers as observations

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
    // Observe provides a list of Eligible upkeep identifiers and the `BlockKey`
    // they were checked at
    Observe(context.Context) (BlockKey, []UpkeepIdentifier, error)
    // MakeReport produces a report from upkeep keys; this is a function on the
    // Observer because on-chain checks will need to be run as well as cache
    // hits on the Observer itself.
    MakeReport(context.Context, []UpkeepKey) ([]byte, int, error)
}
```

The constructor for an `Observer` should take one or more `Config` inputs to
provide internal configuration details.

The constructor should also take a `Coordinator`.

# Polling Observer
A polling observer is a specific implementation of an `Observer` that polls a
chain through a registry every block. To reduce workload, the polling observer
limits the number of upkeeps checked per block based on a sampling ratio where
some random subset of the total upkeeps is checked instead of the entire set.

Each polled upkeep state is added to an observation window that is returned to
the calling function (the `Plugin`). A polling observer must maintain active 
state on pending upkeeps via log events. The `Plugin` indicates to the observer
that an upkeep has been accepted for transmission and the observer must filter
that upkeep from the active window of observations until specific log events
are encountered or a timeout passes for the specific upkeep.

[DIAGRAM](./diagrams/POLLINGOBSERVER.md)

## Ratio
The sampling ratio is determined by the probability that all upkeeps will be
observed by at least `n` nodes over `x` number of blocks. Sampling is to be done
at random over the entire set of upkeeps needing to be polled.

# Log Trigger Observer
A log trigger observer is a modification of the polling observer where a custom
log event triggers polling instead of a block. A custom log event would need to
indicate which upkeep needs to be polled. The specified upkeep is added to a
set of upkeeps needing to be polled and is limited to a polling window. This
provides the same active window of state observations for the upkeep and 
abstracts the differences between a full polling observer and a log trigger
observer.

# Cron Trigger Observer
A cron trigger observer is a further modification of a log trigger observer in
that a custom cron schedule triggers polling on an individual upkeep instead of
logs. In much the same way, a single upkeep would be triggered for polling and
would be limited to a polling window. 
