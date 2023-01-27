package chain

import (
	"fmt"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepKey []byte

func (u UpkeepKey) BlockKeyAndUpkeepID() (types.BlockKey, types.UpkeepIdentifier, error) {
	components := strings.Split(string(u), "|")
	if len(components) == 2 {
		return types.BlockKey(components[0]), types.UpkeepIdentifier(components[1]), nil
	}
	return types.BlockKey(""), types.UpkeepIdentifier(""), fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
}

func (u UpkeepKey) String() string {
	return string(u)
}
