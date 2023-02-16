package chain

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type UpkeepObservation struct {
	BlockKey          BlockKey                 `json:"1"`
	UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
}
