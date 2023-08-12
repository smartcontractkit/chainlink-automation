package ocr2keepers

import (
	"encoding/json"
	"fmt"
	"math/big"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// NOTE: Any change to these values should keep backwards compatibility in mind
// as different nodes would upgrade at different times and would need to
// adhere to each others' limits
const (
	ObservationPerformablesLimit          = 100
	ObservationLogRecoveryProposalsLimit  = 2
	ObservationConditionalsProposalsLimit = 2
	ObservationBlockHistoryLimit          = 256
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
	UpkeepProposals []ocr2keepers.CoordinatedBlockProposal
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

func ValidateAutomationObservation(o AutomationObservation, utg ocr2keepers.UpkeepTypeGetter, wg ocr2keepers.WorkIDGenerator) error {
	// Validate Block History
	if len(o.BlockHistory) > ObservationBlockHistoryLimit {
		return fmt.Errorf("block history length cannot be greater than %d", ObservationBlockHistoryLimit)
	}
	// Block History should not have duplicate block numbers
	seen := make(map[uint64]bool)
	for _, block := range o.BlockHistory {
		if seen[uint64(block.Number)] {
			return fmt.Errorf("block history cannot have duplicate block numbers")
		}
		seen[uint64(block.Number)] = true
	}

	// Validate Performables
	if (len(o.Performable)) > ObservationPerformablesLimit {
		return fmt.Errorf("performable length cannot be greater than %d", ObservationPerformablesLimit)
	}
	seenPerformables := make(map[string]bool)
	for _, res := range o.Performable {
		if err := ValidateCheckResult(res, utg, wg); err != nil {
			return err
		}
		if seenPerformables[res.WorkID] {
			return fmt.Errorf("performable cannot have duplicate workIDs")
		}
		seenPerformables[res.WorkID] = true
	}

	// Validate Proposals
	if (len(o.UpkeepProposals)) >
		(ObservationConditionalsProposalsLimit + ObservationLogRecoveryProposalsLimit) {
		return fmt.Errorf("upkeep proposals length cannot be greater than %d", ObservationConditionalsProposalsLimit+ObservationLogRecoveryProposalsLimit)
	}
	seenProposals := make(map[string]bool)
	for _, proposal := range o.UpkeepProposals {
		if err := ValidateUpkeepProposal(proposal, utg, wg); err != nil {
			return err
		}
		if seenProposals[proposal.WorkID] {
			return fmt.Errorf("proposals cannot have duplicate workIDs")
		}
		seenProposals[proposal.WorkID] = true
	}

	return nil
}

// Validates the check result fields sent within an observation
func ValidateCheckResult(r ocr2keepers.CheckResult, utg ocr2keepers.UpkeepTypeGetter, wg ocr2keepers.WorkIDGenerator) error {
	if r.PipelineExecutionState != 0 || r.Retryable {
		return fmt.Errorf("check result cannot have failed execution state")
	}
	if !r.Eligible || r.IneligibilityReason != 0 {
		return fmt.Errorf("check result cannot be ineligible")
	}
	// UpkeepID is contained [32]byte, no validation needed
	if err := ValidateTriggerExtensionType(r.Trigger, utg(r.UpkeepID)); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}
	if wg(r.UpkeepID, r.Trigger) != r.WorkID {
		return fmt.Errorf("incorrect workID within result")
	}
	if r.GasAllocated == 0 {
		return fmt.Errorf("gas allocated cannot be zero")
	}
	// PerformData is a []byte, no validation needed. Length constraint is handled
	// by maxObservationSize
	uint256Max, _ := big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	if r.FastGasWei.Cmp(big.NewInt(0)) < 0 || r.FastGasWei.Cmp(uint256Max) > 0 {
		return fmt.Errorf("fast gas wei must be in uint256 range")
	}
	if r.LinkNative.Cmp(big.NewInt(0)) < 0 || r.LinkNative.Cmp(uint256Max) > 0 {
		return fmt.Errorf("link native must be in uint256 range")
	}
	return nil
}

// Validate validates the trigger fields, and any extensions if present.
func ValidateTriggerExtensionType(t ocr2keepers.Trigger, ut ocr2keepers.UpkeepType) error {
	switch ut {
	case ocr2keepers.ConditionTrigger:
		if t.LogTriggerExtension != nil {
			return fmt.Errorf("log trigger extension cannot be present for condition upkeep")
		}
	case ocr2keepers.LogTrigger:
		if t.LogTriggerExtension == nil {
			return fmt.Errorf("log trigger extension cannot be empty for log upkeep")
		}
	}
	return nil
}

func ValidateUpkeepProposal(p ocr2keepers.CoordinatedBlockProposal, utg ocr2keepers.UpkeepTypeGetter, wg ocr2keepers.WorkIDGenerator) error {
	// No validation is done on Trigger.BlockNumber and Trigger.BlockHash because those
	// get udpated with a coordinated quorum block
	ut := utg(p.UpkeepID)
	if err := ValidateTriggerExtensionType(p.Trigger, ut); err != nil {
		return err
	}
	if wg(p.UpkeepID, p.Trigger) != p.WorkID {
		return fmt.Errorf("incorrect workID within proposal")
	}
	return nil
}
