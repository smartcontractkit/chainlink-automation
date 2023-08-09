package ocr2keepers

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
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

// AutomationObservation models the proposed actionable decisions made by a single node
type AutomationObservation struct {
	Instructions []instructions.Instruction
	Metadata     map[ObservationMetadataKey]interface{}
	Performable  []ocr2keepers.CheckResult
}

func (observation AutomationObservation) Encode() ([]byte, error) {
	return json.Marshal(observation)
}

func DecodeAutomationObservation(data []byte) (AutomationObservation, error) {
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
	for key := range rawObs.Metadata {
		switch ObservationMetadataKey(key) {
		case BlockHistoryObservationKey:
			// value is a block history type
			// var tmp string
			// var bh ocr2keepers.BlockKey

			// if err := json.Unmarshal(value, &tmp); err != nil {
			// 	return obs, err
			// }
			// parts := strings.Split(tmp, "|")
			// if len(parts) == 0 {
			// 	return obs, fmt.Errorf("%w: %s", ErrWrongDataType, tmp)
			// }
			// if val, ok := big.NewInt(0).SetString(parts[0], 10); !ok {
			// 	return obs, fmt.Errorf("%w: %s", ErrWrongDataType, tmp)
			// } else {
			// 	bh.Number = ocr2keepers.BlockNumber(val.Int64())
			// }
			// metadata[BlockHistoryObservationKey] = ocr2keepers.BlockHistory{bh}
		}
	}

	obs.Instructions = rawObs.Instructions
	obs.Metadata = metadata
	obs.Performable = rawObs.Performable

	return obs, nil
}

func ValidateAutomationObservation(o AutomationObservation) error {
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
		if err := res.Validate(); err != nil {
			return err
		}
	}

	return nil
}
