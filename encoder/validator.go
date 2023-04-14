package encoder

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Validator interface {
	ValidateUpkeepKey(types.UpkeepKey) (bool, error)
	ValidateUpkeepIdentifier(types.UpkeepIdentifier) (bool, error)
	ValidateBlockKey(types.BlockKey) (bool, error)
}
