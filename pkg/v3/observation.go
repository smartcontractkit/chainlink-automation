package ocr2keepers

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
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

// AutomationObservation models the proposed actionable decisions made by a single node
type AutomationObservation struct {
	BlockHistory               ocr2keepers.BlockHistory
	ProposedConditionalSamples []ocr2keepers.UpkeepIdentifier
	ProposedRecoveryLogs       []ocr2keepers.Trigger
	Performable                []ocr2keepers.CheckResult
}

func (observation AutomationObservation) Encode() ([]byte, error) {
	return json.Marshal(observation)
}

func DecodeAutomationObservation(data []byte) (AutomationObservation, error) {
	// TODO: simple decode instead of nested structs
	type raw struct {
		Instructions []instructions.Instruction
		Metadata     map[string]json.RawMessage
		Performable  []ocr2keepers.CheckResult
	}

	var (
		obs    AutomationObservation
		rawObs raw
	)

	if err := json.Unmarshal(data, &rawObs); err != nil {
		return obs, err
	}

	metadata := make(map[ObservationMetadataKey]interface{})
	for key, value := range rawObs.Metadata {
		switch ObservationMetadataKey(key) {
		case BlockHistoryObservationKey:
			// value is a block history type
			var bh ocr2keepers.BlockHistory

			if err := json.Unmarshal(value, &bh); err != nil {
				return obs, err
			}

			metadata[BlockHistoryObservationKey] = bh
		}
	}

	obs.Instructions = rawObs.Instructions
	obs.Metadata = metadata
	obs.Performable = rawObs.Performable

	return obs, nil
}

func ValidateAutomationObservation(o AutomationObservation) error {
	// TODO: add proper validation
	// All values are in range
	// Number of results are witin limits
	for _, in := range o.Instructions {
		if err := instructions.Validate(in); err != nil {
			return err
		}
	}

	for key := range o.Metadata {
		if err := ValidateObservationMetadataKey(key); err != nil {
			return err
		}
	}

	for _, res := range o.Performable {
		if err := ocr2keepers.ValidateCheckResult(res); err != nil {
			return err
		}
	}

	return nil
}
