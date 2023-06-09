package ocrtypes

import (
	"encoding/json"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type AutomationObservation struct {
	Instructions []string
	Metadata     map[string]interface{}
	Performable  []ocr2keepers.CheckResult
}

func (a AutomationObservation) EncodeAutomationObservation() ([]byte, error) {
	return json.Marshal(a)
}

func DecodeAutomationObservation(data []byte) (AutomationObservation, error) {
	var obs AutomationObservation
	err := json.Unmarshal(data, &obs)
	return obs, err
}

type AutomationOutcome struct {
	Instructions []string
	Metadata     map[string]interface{}
	Performable  []ocr2keepers.CheckResult
}

func (a AutomationOutcome) EncodeAutomationOutcome() ([]byte, error) {
	return json.Marshal(a)
}

func DecodeAutomationOutcome(data []byte) (AutomationOutcome, error) {
	var outcome AutomationOutcome
	err := json.Unmarshal(data, &outcome)
	return outcome, err
}
