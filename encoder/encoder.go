package encoder

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Config struct{}

type Encoder interface {
	EncodeReport([]types.UpkeepResult, ...Config) ([]byte, error)
	EncodeUpkeepIdentifier(types.UpkeepResult) (types.UpkeepIdentifier, error)
	KeysFromReport([]byte) ([]types.UpkeepKey, error)
}
