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

// AutomationObservation models the local automation view sent by a single node
// to the network upon which they later get agreement
// NOTE: Any change to this structure should keep backwards compatibility in mind
// as different nodes would upgrade at different times and would need to understand
// each others' observations meanwhile
type AutomationObservation struct {
	// These are the upkeeps that are eligible and should be performed
	Performable []ocr2keepers.CheckResult
	// These are the proposals for upkeeps that need a coordinated block to be checked on
	// The expectation is that once bound to a coordinated block, this goes into performables
	UpkeepProposals []ocr2keepers.CoordinatedProposal
	// This is the block history of the chain from this node's perspective. It sends a
	// few latest blocks to help in block coordination
	BlockHistory ocr2keepers.BlockHistory
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
	// TODO: Only eligible results should be sent and those that have 0 error state
	// TODO: WorkID should be validated
	// TODO: Observations should not have duplicate results
	// TODO: Validate upkeepProposals and blockHistory

	return nil
}
