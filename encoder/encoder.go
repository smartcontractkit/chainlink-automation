package encoder

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Config struct{}

type Encoder interface {
	ValidateUpkeepKey(types.UpkeepKey) (bool, error)
	ValidateUpkeepIdentifier(types.UpkeepIdentifier) (bool, error)
	ValidateBlockKey(types.BlockKey) (bool, error)
	MakeUpkeepKey(types.BlockKey, types.UpkeepIdentifier) types.UpkeepKey
	SplitUpkeepKey(types.UpkeepKey) (types.BlockKey, types.UpkeepIdentifier, error)
	EncodeReport([]types.UpkeepResult, ...Config) ([]byte, error)
	EncodeUpkeepIdentifier(types.UpkeepResult) (types.UpkeepIdentifier, error)
	KeysFromReport([]byte) ([]types.UpkeepKey, error)
	Eligible(types.UpkeepResult) (bool, error)
}
