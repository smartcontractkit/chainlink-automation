# Protocol Testing

The contents of this package are pre-built tools that aid in testing the OCR Automation protocol where one or more nodes
have modified outputs that are in conflict with non-modified nodes. This allows for asserting that the protocol performs
as expected when integrated in a network environment of un-trusted nodes.

## Direct Modifiers

Direct modifiers apply data changes directly before or directly after encoding of either observations, outcomes, or
reports. This output modification strategy allows for encoding changes outside the scope of strict Golang types as well
as simple value modifications directly on the output type.

The subpackage `modify` contains the general modifier structure as well as some pre-built modifiers and collection
constants. There are two modifier variants:

- `Modify`: takes a modifier input such as `AsObservation` or `AsOutcome` which apply type assertions on the subsequent modifiers
- `ModifyBytes`: takes a slice of `MapModifier` where key/value pairs are provided to modifiers

### Modify

Use the `Modify` type when applying modifications directly to a type such as `AutomationOutcome`. Multiple usage
examples are located in `modify/defaults.go`. To write new modifiers, follow the pattern below:

```
// WithPerformableBlockAs adds the provided block number within the scope of a PerformableModifier function
func WithPerformableBlockAs(block types.BlockNumber) PerformableModifier {
	return func(performables []types.CheckResult) []types.CheckResult {
        // the block number in scope is applied to all performables
		for _, performable := range performables {
			performable.Trigger.BlockNumber = block
		}

        // the modified performables are returned back to the observation or outcome
		return performables
	}
}
```

Use this modifier in multiple typed modifiers as a composible function:

```
// use the above function to modify performables in observations
Modify(
    "set all performables to block 1",
    AsObservation(
        WithPerformableBlockAs(types.BlockNumber(1))))

// use the above function to modify performables in outcomes
Modify(
    "set all performables to block 1",
    AsOutcome(
        WithPerformableBlockAs(types.BlockNumber(1))))
```

### ModifyBytes

This modify function can be used to change values directly in a json encoded output. Instead of operating on direct
types like `AutomationOutcome`, the json input is split into key/value pairs before being passed to subsequent custom
modifiers. Write a new modifier using the following pattern:

```
// WithModifyKeyValue is a generic key/value modifier where a key and modifier function are provided and the function
// recursively searches the json path for the provided key and modifies the value when the key is found.
func WithModifyKeyValue(key string, fn ValueModifierFunc) MapModifier {
	return func(ctx context.Context, values map[string]interface{}, err error) (map[string]interface{}, error) {
		return recursiveModify(key, "root", fn, values), err
	}
}
```

Use this modifier as a generic key/value modifer for arbitrary json structures:

```
ModifyBytes(
    "set block value to very large number as string",
    WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
        return "98989898989898989898989898989898989898989898"
    }))
```

## Indirect Modifiers

In many cases, data modifications must be applied BEFORE an observation or outcome can be constructed. These types of
cases might include repeated proposals where state between rounds might need to be tracked and specific data needs to be
captured and re-broadcast where the unmodified protocol wouldn't. 

Specifics TBD