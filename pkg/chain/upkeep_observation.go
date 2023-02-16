package chain

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepObservation struct {
	BlockKey          BlockKey                 `json:"blockKey"`
	UpkeepIdentifiers []types.UpkeepIdentifier `json:"upkeepIdentifiers"`
}
