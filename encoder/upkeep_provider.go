package encoder

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepProvider interface {
	MakeUpkeepKey(types.BlockKey, types.UpkeepIdentifier) types.UpkeepKey
	SplitUpkeepKey(types.UpkeepKey) (types.BlockKey, types.UpkeepIdentifier, error)
}
