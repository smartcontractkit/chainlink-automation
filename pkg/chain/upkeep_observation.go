package chain

import (
	"fmt"
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrInvalidBlockKey         = fmt.Errorf("invalid block key")
	ErrInvalidUpkeepIdentifier = fmt.Errorf("invalid upkeep identifier")
)

// TODO (AUTO-2014), find a better place for these concrete types than chain package
type UpkeepObservation struct {
	BlockKey          BlockKey                 `json:"1"`
	UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
}

func (uo UpkeepObservation) Validate() error {
	bl, ok := big.NewInt(0).SetString(uo.BlockKey.String(), 10)
	if !ok {
		return ErrBlockKeyNotParsable
	}
	if bl.String() != uo.BlockKey.String() {
		return ErrInvalidBlockKey

	}

	for _, ui := range uo.UpkeepIdentifiers {
		uiInt, ok := ui.BigInt()
		if !ok {
			return ErrUpkeepKeyNotParsable
		}

		if uiInt.String() != string(ui) {
			return ErrInvalidUpkeepIdentifier
		}
	}

	return nil
}
