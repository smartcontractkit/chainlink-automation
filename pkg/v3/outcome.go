package ocr2keepers

import (
	"encoding/json"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// AutomationOutcome represents agreed upon state by the network, derived from
// a collection of AutomationObservations with applied quorum thresholds
// NOTE: Any change to this structure should keep backwards compatibility in mind
// as different nodes would upgrade at different times and would need to understand
// each others' outcome meanwhile
type AutomationOutcome struct {
	// These are the upkeeps that got quorum that they should be performed on chain
	// These require quorum of f+1 nodes
	AgreedPerformables []ocr2keepers.CheckResult
	// These are the proposals with a coordinated block that should be run through the
	// check pipeline. The proposals remain valid for a range of rounds where they do
	// not get tied to a new coordinated block in order to give check pipeline enough
	// time to run asynchronously
	// Quorum of f+1 is only applied on the blockNumber and blockHash of the proposal
	// rest of the fields can be manipulated by malicious nodes
	SurfacedProposals [][]ocr2keepers.CoordinatedBlockProposal
}

// DecodeAutomationOutcome decodes an AutomationOutcome from an encoded array
// of bytes. Possible errors come from the encoding/json package
func DecodeAutomationOutcome(data []byte) (AutomationOutcome, error) {
	ao := AutomationOutcome{}
	err := json.Unmarshal(data, &ao)
	return ao, err
}

// ValidateAutomationOutcome validates individual values in an AutomationOutcome
func ValidateAutomationOutcome(o AutomationOutcome) error {
	// TODO: Validate sizes of AgreedPerformables, AgreedProposals
	// TODO: Validate AgreedPerformables and AgreedProposals
	return nil
}

// Encode produces a json encoded array of bytes. Possible errors come from the
// encoding/json package
func (outcome AutomationOutcome) Encode() ([]byte, error) {
	return json.Marshal(outcome)
}
