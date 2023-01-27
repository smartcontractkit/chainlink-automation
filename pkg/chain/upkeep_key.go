package chain

import (
	"encoding/json"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"strings"
)

type UpkeepKey []byte

func (u UpkeepKey) BlockKeyAndUpkeepID() (types.BlockKey, types.UpkeepIdentifier) {
	components := strings.Split(string(u), "|")
	if len(components) == 2 {
		return types.BlockKey(components[0]), types.UpkeepIdentifier(components[1])
	}
	return types.BlockKey(""), types.UpkeepIdentifier("")
}

func (u UpkeepKey) String() string {
	return string(u)
}

func (u *UpkeepKey) UnmarshalJSON(b []byte) error {
	var key []uint8
	if err := json.Unmarshal(b, &key); err != nil {
		return err
	}

	*u = UpkeepKey([]byte(key))
	return nil
}
