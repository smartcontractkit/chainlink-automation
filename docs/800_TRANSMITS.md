# Report Coordinator

There are two methods to be supported for confirming transmits made it to chain.

1. check that any key from a `Report` has a perform log received by the node
2. (future) check that an OCR transmit log was received

This component is intended to encompass the current functionality of a filter.

## Interface

The following satisfies the former for now until future contract changes are
complete.

```
type Coordinator interface {
    IsPending(UpkeepKey) (bool, error)
    // Accept adds all upkeep keys to an internal filter
    Accept([]UpkeepKey) error
    // IsTransmissionConfirmed is a temporary function that checks if a log has
    // been received for a particular key
    IsTransmissionConfirmed(UpkeepKey) bool
}
```
