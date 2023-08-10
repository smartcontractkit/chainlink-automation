package ocr2keepers

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type ObservationMetadataKey string

const (
	BlockHistoryObservationKey     ObservationMetadataKey = "blockHistory"
	SampleProposalObservationKey   ObservationMetadataKey = "sampleProposals"
	RecoveryProposalObservationKey ObservationMetadataKey = "recoveryProposals"
)

var (
	ErrInvalidMetadataKey = fmt.Errorf("invalid metadata key")
	ErrWrongDataType      = fmt.Errorf("wrong data type")
	ErrBlockNotAvailable  = fmt.Errorf("coordinated block not available in outcome")
)

func ValidateObservationMetadataKey(key ObservationMetadataKey) error {
	switch key {
	case BlockHistoryObservationKey:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidMetadataKey, key)
	}
}

// AutomationObservation models the proposed actionable decisions sent by a single node
// to the network upon which they later get agreement
// NOTE: Any change to this structure should keep backwards compatibility in mind
// as different nodes would upgrade at different times and would need to understand
// each other's observations in the meantime
type AutomationObservation struct {
	UpkeepProposals []ocr2keepers.CoordinatedProposal
	BlockHistory    ocr2keepers.BlockHistory
	Performable     []ocr2keepers.CheckResult
}

func (observation AutomationObservation) Encode() ([]byte, error) {
	return json.Marshal(observation)
}

func DecodeAutomationObservation(data []byte) (AutomationObservation, error) {
	ao := AutomationObservation{}
	err := json.Unmarshal(data, &ao)
	return ao, err
}

func ValidateAutomationObservation(o AutomationObservation) error {
	// TODO: Validate sizes of upkeepProposals, BlockHistory, Performables
	for _, res := range o.Performable {
		if err := res.Validate(); err != nil {
			return err
		}
	}
	// TODO: Validate upkeepProposals and blockHistory

	return nil
}
