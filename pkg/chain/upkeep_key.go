package chain

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepKey []byte

func (u UpkeepKey) BlockKeyAndUpkeepID() (types.BlockKey, types.UpkeepIdentifier, error) {
	components := strings.Split(u.String(), "|")
	if len(components) == 2 {
		return types.BlockKey(components[0]), types.UpkeepIdentifier(components[1]), nil
	}
	return types.BlockKey(""), types.UpkeepIdentifier(""), fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
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
