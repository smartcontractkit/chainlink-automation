package ocr2keepers

import (
	"encoding/json"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
)

// AutomationObservation models the proposed actionable decisions made by a single node
type AutomationObservation struct {
	Instructions []instruction.Instruction
	Metadata     map[string]interface{}
	Performable  []ocr2keepers.CheckResult
}

func (observation AutomationObservation) Encode() ([]byte, error) {
	return json.Marshal(observation)
}

func DecodeAutomationObservation(data []byte) (AutomationObservation, error) {
	var obs AutomationObservation
	err := json.Unmarshal(data, &obs)
	return obs, err
}

// AutomationOutcome represents decisions proposed by a single node based on observations.
type AutomationOutcome struct {
	Instructions []instruction.Instruction
	Metadata     map[string]interface{}
	Performable  []ocr2keepers.CheckResult
}

func (outcome AutomationOutcome) Encode() ([]byte, error) {
	return json.Marshal(outcome)
}

func DecodeAutomationOutcome(data []byte) (AutomationOutcome, error) {
	var outcome AutomationOutcome
	err := json.Unmarshal(data, &outcome)
	return outcome, err
}
