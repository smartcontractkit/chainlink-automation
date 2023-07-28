package ocr2keepers

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
)

type OutcomeMetadataKey string

const (
	CoordinatedBlockOutcomeKey     OutcomeMetadataKey = "coordinatedBlock"
	CoordinatedRecoveryProposalKey OutcomeMetadataKey = "coordinatedRecoveryProposals"
	CoordinatedSamplesProposalKey  OutcomeMetadataKey = "coordinatedSampleProposals"
)

func ValidateOutcomeMetadataKey(key OutcomeMetadataKey) error {
	switch key {
	case CoordinatedBlockOutcomeKey, CoordinatedRecoveryProposalKey, CoordinatedSamplesProposalKey:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidMetadataKey, key)
	}
}

// DecodeAutomationOutcome decodes an AutomationOutcome from an encoded array
// of bytes. Possible errors come from the encoding/json package
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

// ValidateAutomationOutcome validates individual values in an AutomationOutcome
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

// AutomationOutcome represents decisions proposed by a single node based on
// observations
type AutomationOutcome struct {
	BasicOutcome
	Instructions []instructions.Instruction
	History      []BasicOutcome
	NextIdx      int
}

// Encode produces a json encoded array of bytes. Possible errors come from the
// encoding/json package
func (outcome AutomationOutcome) Encode() ([]byte, error) {
	return json.Marshal(outcome)
}

// RecoveryProposals is a helper function for accessing recovery proposals from
// an outcome. This accessor does not search through outcome history.
func (outcome AutomationOutcome) RecoveryProposals() ([]ocr2keepers.CoordinatedProposal, error) {
	var (
		ok           bool
		rawProposals interface{}
		proposals    []ocr2keepers.CoordinatedProposal
	)

	// if recoverable items are in outcome, proceed with values
	if rawProposals, ok = outcome.Metadata[CoordinatedRecoveryProposalKey]; !ok {
		return nil, nil
	}

	// proposals are trigger ids
	if proposals, ok = rawProposals.([]ocr2keepers.CoordinatedProposal); !ok {
		return nil, fmt.Errorf("%w: coordinated proposals are not of type `CoordinatedProposal`", ErrWrongDataType)
	}

	return proposals, nil
}

// LatestCoordinatedBlock is a helper function that provides easy access to the
// latest block coordinated by all nodes
func (outcome AutomationOutcome) LatestCoordinatedBlock() (ocr2keepers.BlockKey, error) {
	// get latest coordinated block
	// by checking latest outcome first and then looping through the history
	var (
		rawBlock       interface{}
		blockAvailable bool
		block          ocr2keepers.BlockKey
		ok             bool
	)

	if rawBlock, ok = outcome.Metadata[CoordinatedBlockOutcomeKey]; !ok {
		// values from sorted history are newest to oldest
		// if a coordinated block is encountered, exit the loop with the value
		for _, h := range outcome.SortedHistory() {
			if rawBlock, ok = h.Metadata[CoordinatedBlockOutcomeKey]; !ok {
				continue
			}

			blockAvailable = true

			break
		}
	} else {
		blockAvailable = true
	}

	// a latest block isn't available
	if !blockAvailable {
		return block, ErrBlockNotAvailable
	}

	if block, ok = rawBlock.(ocr2keepers.BlockKey); !ok {
		return block, fmt.Errorf("%w: coordinated block value not of type `BlockKey`", ErrWrongDataType)
	}

	return block, nil
}

// SortedHistory is a helper function for accessing the history ring buffer.
// Values returned are sorted newest to oldest
func (outcome AutomationOutcome) SortedHistory() []BasicOutcome {
	slice := make([]BasicOutcome, len(outcome.History))

	// idx is the index of the last inserted value
	idx := outcome.NextIdx - 1
	if idx < 0 {
		idx = len(outcome.History) - 1
	}

	for x := 0; x < len(outcome.History); x++ {
		slice[x] = outcome.History[idx]

		// run in reverse order to access newest to oldest
		idx--

		if idx < 0 {
			idx = len(outcome.History) - 1
		}
	}

	return slice
}

// UpkeepIdentifiers extracts all upkeep identifiers from the outcome but does
// not access outcome history
func (outcome AutomationOutcome) UpkeepIdentifiers() ([]ocr2keepers.UpkeepIdentifier, error) {
	var (
		ok           bool
		rawProposals interface{}
		proposals    []ocr2keepers.UpkeepIdentifier
	)

	// if recoverable items are in outcome, proceed with values
	if rawProposals, ok = outcome.Metadata[CoordinatedSamplesProposalKey]; !ok {
		return nil, fmt.Errorf("no value for key: %s", CoordinatedRecoveryProposalKey)
	}

	// proposals are upkeep ids
	if proposals, ok = rawProposals.([]ocr2keepers.UpkeepIdentifier); !ok {
		return nil, fmt.Errorf("%w: coordinated proposals are not of type `CoordinatedProposal`", ErrWrongDataType)
	}

	return proposals, nil
}

// BasicOutcome is the common structure of an AutomationOutcome and outcome
// history elements
type BasicOutcome struct {
	Metadata    map[OutcomeMetadataKey]interface{}
	Performable []ocr2keepers.CheckResult
}

// UnmarshalJSON implements the json unmarshaller interface and decodes a
// BasicOutcome from bytes. If a metadata key is unrecognized, the mapped key
// and value are skipped
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
