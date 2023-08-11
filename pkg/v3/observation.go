package ocr2keepers

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

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
		if err := ValidateCheckResult(res); err != nil {
			return err
		}
	}
	// TODO: Only eligible results should be sent and those that have 0 error state
	// TODO: WorkID should be validated
	// TODO: Observations should not have duplicate results
	// proposals should not have dplicate workIDs
	// blockHistory should not have duplicate numbers
	// TODO: Validate upkeepProposals and blockHistory

	return nil
}

// Validate validates the check result fields
func ValidateCheckResult(r ocr2keepers.CheckResult) error {
	if r.PipelineExecutionState == 0 && r.Retryable {
		return fmt.Errorf("check result cannot have successful execution state and be retryable")
	}
	if r.PipelineExecutionState == 0 {
		if r.Eligible && r.IneligibilityReason != 0 {
			return fmt.Errorf("check result cannot be eligible and have an ineligibility reason")
		}
		if r.IneligibilityReason == 0 && !r.Eligible {
			return fmt.Errorf("check result cannot be ineligible and have no ineligibility reason")
		}
		if r.Eligible {
			// TODO: This should be checked only if eligible
			if r.GasAllocated == 0 {
				return fmt.Errorf("gas allocated cannot be zero")
			}
			// TODO: add validation for upkeepType and presence of trigger extension
			// TODO: add range validation on linkNative and fasGas (uint256)
		}
	}
	return nil
}

// Validate validates the trigger fields, and any extensions if present.
func ValidateTrigger(t ocr2keepers.Trigger) error {
	if t.BlockNumber == 0 {
		return fmt.Errorf("block number cannot be zero")
	}
	if len(t.BlockHash) == 0 {
		return fmt.Errorf("block hash cannot be empty")
	}

	if t.LogTriggerExtension != nil {
		if err := ValidateLogTriggerExtension(*t.LogTriggerExtension); err != nil {
			return fmt.Errorf("log trigger extension invalid: %w", err)
		}
	}

	return nil
}

// Validate validates the log trigger extension fields.
// NOTE: not checking block hash or block number because they might not be available (e.g. ReportedUpkeep)
func ValidateLogTriggerExtension(e ocr2keepers.LogTriggerExtension) error {
	if len(e.TxHash) == 0 {
		return fmt.Errorf("log transaction hash cannot be empty")
	}
	if e.Index == 0 {
		return fmt.Errorf("log index cannot be zero")
	}

	return nil
}
