package chain

import (
	"encoding/json"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// TODO (AUTO-2014), find a better place for these concrete types than chain package
type UpkeepObservation struct {
	BlockKey          BlockKey                 `json:"1"`
	UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
}

type upkeepObservation UpkeepObservation

func (u *UpkeepObservation) UnmarshalJSON(b []byte) error {
	var upkeep upkeepObservation
	if err := json.Unmarshal(b, &upkeep); err != nil {
		return err
	}

	if !u.isValid(upkeep) {
		return ErrUpkeepObservationNotParsable
	}

	*u = UpkeepObservation(upkeep)

	return nil
}

func (u *UpkeepObservation) isValid(v upkeepObservation) bool {
	return strings.TrimSpace(v.BlockKey.String()) != ""
}
