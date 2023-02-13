package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepKey []byte

// NewUpkeepKey is the constructor of UpkeepKey
func NewUpkeepKey(block, id *big.Int) UpkeepKey {
	return UpkeepKey(fmt.Sprintf("%s%s%s", block, separator, id))
}

func NewUpkeepKeyFromBlockAndID(block types.BlockKey, id types.UpkeepIdentifier) UpkeepKey {
	return UpkeepKey(fmt.Sprintf("%s%s%s", string(block), separator, string(id)))
}

func (u UpkeepKey) BlockKeyAndUpkeepID() (types.BlockKey, types.UpkeepIdentifier, error) {
	components := strings.Split(u.String(), "|")
	if len(components) != 2 {
		return nil, nil, fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
	}

	return BlockKey(components[0]), types.UpkeepIdentifier(components[1]), nil
}

func (u UpkeepKey) String() string {
	return string(u)
}

func (u *UpkeepKey) UnmarshalJSON(b []byte) error {
	var key []uint8
	if err := json.Unmarshal(b, &key); err != nil {
		return err
	}

	if !u.isValid(string(key)) {
		return ErrUpkeepKeyNotParsable
	}

	*u = UpkeepKey(key)

	return nil
}

func (u *UpkeepKey) isValid(v string) bool {
	if strings.Count(v, separator) != 1 {
		return false
	}

	components := strings.Split(v, separator)
	if len(components) != 2 {
		return false
	}

	if components[0] == "" || components[1] == "" {
		return false
	}

	return true
}
