package v1

import (
	"fmt"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type upkeepProvider struct{}

// MakeUpkeepKey creates a new types.UpkeepKey from a types.BlockKey and a types.UpkeepIdentifier
func (p upkeepProvider) MakeUpkeepKey(blockKey types.BlockKey, upkeepIdentifier types.UpkeepIdentifier) types.UpkeepKey {
	return chain.UpkeepKey(fmt.Sprintf("%s%s%s", blockKey, separator, string(upkeepIdentifier)))
}

// SplitUpkeepKey splits a types.UpkeepKey into its constituent types.BlockKey and types.UpkeepIdentifier parts
func (p upkeepProvider) SplitUpkeepKey(upkeepKey types.UpkeepKey) (types.BlockKey, types.UpkeepIdentifier, error) {
	if upkeepKey == nil {
		return nil, nil, fmt.Errorf("%w: missing data in upkeep key", errUpkeepKeyNotParsable)
	}
	components := strings.Split(upkeepKey.String(), separator)
	if len(components) != 2 {
		return nil, nil, fmt.Errorf("%w: missing data in upkeep key", errUpkeepKeyNotParsable)
	}

	return chain.BlockKey(components[0]), types.UpkeepIdentifier(components[1]), nil
}
