package ocr2keepers

import (
	"strings"
)

type UpkeepIdentifier []byte

type BlockKey []byte

type UpkeepKey []byte

type UpkeepResult interface{}

func upkeepKeysToString(keys []UpkeepKey) string {
	keysStr := make([]string, len(keys))
	for i, key := range keys {
		keysStr[i] = string(key)
	}

	return strings.Join(keysStr, ", ")
}
