package ocr2keepers

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
)

type ObservationMetadataKey string
type OutcomeMetadataKey string

const (
	BlockHistoryObservationKey ObservationMetadataKey = "blockHistory"
	CoordinatedBlockOutcomeKey OutcomeMetadataKey     = "coordinatedBlock"
)

var (
	ErrInvalidMetadataKey = fmt.Errorf("invalid metadata key")
)

func ValidateObservationMetadataKey(key ObservationMetadataKey) error {
	switch key {
	case BlockHistoryObservationKey:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidMetadataKey, key)
	}
}

func ValidateOutcomeMetadataKey(key OutcomeMetadataKey) error {
	switch key {
	case CoordinatedBlockOutcomeKey:
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

// AutomationOutcome represents decisions proposed by a single node based on observations.
type AutomationOutcome struct {
	BasicOutcome
	Instructions []instructions.Instruction
	History      []BasicOutcome
	NextIdx      int
}

type BasicOutcome struct {
	Metadata    map[OutcomeMetadataKey]interface{}
	Performable []ocr2keepers.CheckResult
}

func (bo *BasicOutcome) UnmarshalJSON(b []byte) error {
	type raw struct {
		Metadata    map[string]json.RawMessage
		Performable []ocr2keepers.CheckResult
	}

	var rawOutcome raw

	if err := json.Unmarshal(b, &rawOutcome); err != nil {
		return err
	}

	metadata := make(map[OutcomeMetadataKey]interface{})
	for key, value := range rawOutcome.Metadata {
		switch OutcomeMetadataKey(key) {
		case CoordinatedBlockOutcomeKey:
			// value is a block history type
			var bk ocr2keepers.BlockKey

			if err := json.Unmarshal(value, &bk); err != nil {
				return err
			}

			metadata[CoordinatedBlockOutcomeKey] = bk
		}
	}

	*bo = BasicOutcome{
		Metadata:    metadata,
		Performable: rawOutcome.Performable,
	}

	return nil
}

func (outcome AutomationOutcome) Encode() ([]byte, error) {
	return json.Marshal(outcome)
}

func DecodeAutomationOutcome(data []byte) (AutomationOutcome, error) {
	type raw struct {
		Instructions []instructions.Instruction
		History      []BasicOutcome
		NextIdx      int
	}

	var (
		outcome         AutomationOutcome
		rawOutcome      raw
		rawBasicOutcome BasicOutcome
	)

	if err := json.Unmarshal(data, &rawOutcome); err != nil {
		return outcome, err
	}

	if err := json.Unmarshal(data, &rawBasicOutcome); err != nil {
		return outcome, err
	}

	return AutomationOutcome{
		BasicOutcome: BasicOutcome{
			Metadata:    rawBasicOutcome.Metadata,
			Performable: rawBasicOutcome.Performable,
		},
		Instructions: rawOutcome.Instructions,
		History:      rawOutcome.History,
		NextIdx:      rawOutcome.NextIdx,
	}, nil
}

func ValidateAutomationOutcome(o AutomationOutcome) error {
	for _, in := range o.Instructions {
		if err := instructions.Validate(in); err != nil {
			return err
		}
	}

	for key := range o.Metadata {
		if err := ValidateOutcomeMetadataKey(key); err != nil {
			return err
		}
	}

	for _, res := range o.Performable {
		if err := ocr2keepers.ValidateCheckResult(res); err != nil {
			return err
		}
	}

	if o.NextIdx > len(o.History) {
		return fmt.Errorf("invalid ring buffer index: %d for history length %d", o.NextIdx, len(o.History))
	}

	for _, h := range o.History {
		for key := range h.Metadata {
			if err := ValidateOutcomeMetadataKey(key); err != nil {
				return err
			}
		}

		for _, res := range h.Performable {
			if err := ocr2keepers.ValidateCheckResult(res); err != nil {
				return err
			}
		}
	}

	return nil
}
