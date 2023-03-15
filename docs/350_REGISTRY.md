# What is a registry?

A `Registry` should be responsible for contract specific behavior and consolidate
the outcomes of that specific behavior into results that an `Observer` can 
consume.

To maintain an abstraction between an `Observer`, `Encoder`, and a `Registry`, 
the return type of a `Registry` should be a wrapper for an `interface{}` type.

```
type UpkeepResult interface{}
```

Setting this type specifically allows for distinct types when describing the
output of one function as the input of another, but provides the necessary type
abstraction from one registry version to another. Each `Registry` / `Encoder`
pairing should be responsible for its own type assertions.

## Registry Interfaces

Different `Observer` types can require specific functionality from any 
particular registry. The folling defines the composability of a `Registry` and
how that maps to an `Observer` type.

### Must Implement

All registries MUST implement the following interface which performs all checks
required to determine if an upkeep is eligible to be performed or not. The
initial intent is to not make a breaking change to the existing registry
implementation by making a major interface change. The only change here is
to return the above described type which will just be a wrapper for the existing
type. An observer does not need to know any of the details of the result. It 
only needs to be able to pass the result along to an `Encoder`.

```
type Registry interface {
    CheckUpkeep(context.Context, ...UpkeepKey) ([]UpkeepResult, error)
}
```

### May Implement

The following is an example of a `Registry` interface implementation that a 
polling `Observer` would require. 

```
// PollingRegistry is intended to provide functions required by a polling
// observer.
type PollingRegistry interface {
    // HeadTicker provides blocks at a trigger
    HeadTicker() chan BlockKey
    // GetActiveUpkeepIDs provides a complete list of ids required to be polled
    GetActiveUpkeepIDs(context.Context) ([]UpkeepIdentifier, error)
}
```