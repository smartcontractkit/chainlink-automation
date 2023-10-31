package modify

import (
	"context"
	"encoding/json"
	"fmt"
)

type NamedByteModifier func(context.Context, []byte, error) (string, []byte, error)
type MapModifier func(context.Context, map[string]interface{}, error) (map[string]interface{}, error)
type ValueModifierFunc func(string, interface{}) interface{}

// WithModifyKeyValue recursively operates on all key-value pairs in the provided map and applies the provided modifier
// function if the key matches. The path provided to the modifier function starts with `root` and is appended with every
// key encountered in the tree. ex: `root.someKey.anotherKey`.
func WithModifyKeyValue(key string, fn ValueModifierFunc) MapModifier {
	return func(ctx context.Context, values map[string]interface{}, err error) (map[string]interface{}, error) {
		return recursiveModify(key, "root", fn, values), err
	}
}

// ModifyBytes deconstructs provided bytes into a map[string]interface{} and passes the decoded map to provided
// modifiers. The final modified map is re-encoded as bytes and returned by the modifier function.
func ModifyBytes(name string, modifiers ...MapModifier) NamedByteModifier {
	return func(ctx context.Context, bytes []byte, err error) (string, []byte, error) {
		var values map[string]interface{}

		if err := json.Unmarshal(bytes, &values); err != nil {
			return name, bytes, err
		}

		for _, modifier := range modifiers {
			values, err = modifier(ctx, values, err)
		}

		bytes, err = json.Marshal(values)

		return name, bytes, err
	}
}

func recursiveModify(key, path string, mod ValueModifierFunc, values map[string]interface{}) map[string]interface{} {
	for mapKey, mapValue := range values {
		newPath := fmt.Sprintf("%s.%s", path, mapKey)

		switch nextValues := mapValue.(type) {
		case map[string]interface{}:
			values[key] = recursiveModify(key, newPath, mod, nextValues)
		case []interface{}:
			for idx, arrayValue := range nextValues {
				newPath = fmt.Sprintf("%s[%d]", newPath, idx)

				if mappedArray, ok := arrayValue.(map[string]interface{}); ok {
					nextValues[idx] = recursiveModify(key, newPath, mod, mappedArray)
				}
			}
		default:
			if mapKey == key {
				values[key] = mod(newPath, mapValue)
			}
		}
	}

	return values
}
