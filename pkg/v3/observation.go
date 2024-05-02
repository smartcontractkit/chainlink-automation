package ocr2keepers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

// NOTE: Any change to these values should keep backwards compatibility in mind
// as different nodes would upgrade at different times and would need to
// adhere to each others' limits
const (
	ObservationPerformablesLimit          = 50
	ObservationLogRecoveryProposalsLimit  = 5
	ObservationConditionalsProposalsLimit = 5
	ObservationBlockHistoryLimit          = 256

	// MaxObservationLength applies a limit to the total length of bytes in an
	// observation. NOTE: This is derived from a limit of 10000 on performData
	// which is guaranteed onchain
	MaxObservationLength = 1_000_000
)

var uint256Max, _ = big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

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

func (observation AutomationObservation) Length() int {
	numberOfFields := 3 // should not be counted, used to coordinate values
	nullFieldChars := 4 // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1
	performablesNameAndQuotes := 13
	upkeepProposalsNameAndQuotes := 17
	blockHistoryNameAndQuotes := 14

	fieldValues := 0
	if observation.Performable == nil {
		fieldValues += nullFieldChars
	} else {
		fieldValues += performablesLength(observation.Performable)
	}

	if observation.UpkeepProposals == nil {
		fieldValues += nullFieldChars
	} else {
		fieldValues += upkeepProposalsLength(observation.UpkeepProposals)
	}

	if observation.BlockHistory == nil {
		fieldValues += nullFieldChars
	} else {
		fieldValues += blockHistoryLength(observation.BlockHistory)
	}

	return objectBraces + fieldSeparators + numberOfColons + performablesNameAndQuotes + upkeepProposalsNameAndQuotes + blockHistoryNameAndQuotes + fieldValues
}

func performablesLength(results []ocr2keepers.CheckResult) int {
	if len(results) == 0 {
		return 2
	} else {
		performablesLength := 0

		numberOfFields := len(results) // should not be counted, used to coordinate values, we don't include retry interval

		objectBraces := 2
		fieldSeparators := numberOfFields - 1

		for _, result := range results {
			performablesLength += performableLength(result)
		}

		return objectBraces + fieldSeparators + performablesLength
	}
}

func performableLength(result ocr2keepers.CheckResult) int {
	numberOfFields := 11 // should not be counted, used to coordinate values, we don't include retry interval
	nullFieldChars := 4  // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1

	pipelineExecutionStateNameAndQuotes := 24
	retryablenameAndQuotes := 11
	eligibleNameAndQuotes := 10
	ineligibilityReasonNameAndQuotes := 21
	upkeepIDNameAndQuotes := 10
	triggerNameAndQuotes := 9
	workIDNameAndQuotes := 8
	gasAllocatedNameAndQuotes := 14
	performDataNameAndQuotes := 13
	fastGasWeiNameAndQuotes := 12
	linkNativeNameAndQuotes := 12

	valueSizes := 0
	valueSizes += uint8Length(result.PipelineExecutionState)
	valueSizes += boolLength(result.Retryable)
	valueSizes += boolLength(result.Eligible)
	valueSizes += uint8Length(result.IneligibilityReason)
	valueSizes += 2 + byte32Length(result.UpkeepID) + len(result.UpkeepID) - 1 // 2 for brackets, length of bytes, length-1 for commas

	valueSizes += triggerLength(result.Trigger)

	valueSizes += 2 + len(result.WorkID)
	valueSizes += uint64Length(result.GasAllocated)

	if result.PerformData == nil {
		valueSizes += nullFieldChars
	} else {
		valueSizes += 2 + len(hex.EncodeToString(result.PerformData))
	}

	if result.FastGasWei == nil {
		valueSizes += nullFieldChars
	} else {
		valueSizes += len(result.FastGasWei.String()) // quotes aren't included in big int json
	}

	if result.LinkNative == nil {
		valueSizes += nullFieldChars
	} else {
		valueSizes += len(result.LinkNative.String()) // quotes aren't included in big int json
	}

	return objectBraces + numberOfColons + fieldSeparators + pipelineExecutionStateNameAndQuotes + retryablenameAndQuotes +
		eligibleNameAndQuotes + ineligibilityReasonNameAndQuotes + upkeepIDNameAndQuotes + triggerNameAndQuotes + workIDNameAndQuotes +
		gasAllocatedNameAndQuotes + performDataNameAndQuotes + fastGasWeiNameAndQuotes + linkNativeNameAndQuotes + valueSizes
}

func triggerLength(t ocr2keepers.Trigger) int {
	numberOfFields := 3 // should not be counted, used to coordinate values
	nullFieldChars := 4 // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1

	blockNumberNameAndQuotes := 13
	blockHashNameAndQuotes := 11
	logTriggerExtensionNameAndQuotes := 21

	valueSizes := 0

	valueSizes += uint64Length(uint64(t.BlockNumber))
	valueSizes += 2 + byte32Length(t.BlockHash) + len(t.BlockHash) - 1 // 2 for brackets, length of bytes, length-1 for commas

	if t.LogTriggerExtension == nil {
		valueSizes += nullFieldChars
	} else {
		valueSizes += logTriggerExtensionLength(t.LogTriggerExtension)
	}

	return objectBraces + numberOfColons + fieldSeparators + blockNumberNameAndQuotes + blockHashNameAndQuotes + logTriggerExtensionNameAndQuotes + valueSizes
}

func logTriggerExtensionLength(extension *ocr2keepers.LogTriggerExtension) int {
	numberOfFields := 4 // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1

	txHashNameAndQuotes := 8
	indexNameAndQuotes := 7
	blockHashNameAndQuotes := 11
	blockNumberNameAndQuotes := 13

	valueSizes := 0

	valueSizes += 2 + byte32Length(extension.TxHash) + len(extension.TxHash) - 1 // 2 for brackets, length of bytes, length-1 for commas
	valueSizes += uint32Length(extension.Index)
	valueSizes += uint64Length(uint64(extension.BlockNumber))
	valueSizes += 2 + byte32Length(extension.BlockHash) + len(extension.BlockHash) - 1 /// 2 for brackets, length of bytes, length-1 for commas

	return objectBraces + numberOfColons + fieldSeparators + txHashNameAndQuotes + indexNameAndQuotes + blockNumberNameAndQuotes + blockHashNameAndQuotes + valueSizes

}

func byteSliceLength(b []byte) int {
	len := 0

	for _, x := range b {
		len += uint8Length(x)
	}

	return len
}

func byte32Length(b [32]byte) int {
	return byteSliceLength(b[:])
}

func uint64Length(i uint64) int {
	return len(strconv.FormatUint(i, 10))
}

func uint32Length(i uint32) int {
	return len(strconv.FormatUint(uint64(i), 10))
}

func uint8Length(i uint8) int {
	if i < 10 {
		return 1
	} else if i < 100 {
		return 2
	}
	return 3
}

func boolLength(b bool) int {
	if b {
		return 4
	}
	return 5
}

func upkeepProposalsLength(proposals []ocr2keepers.CoordinatedBlockProposal) int {
	if len(proposals) == 0 {
		return 2
	} else {
		proposalsLength := 0

		numberOfFields := len(proposals) // should not be counted, used to coordinate values, we don't include retry interval

		objectBraces := 2
		fieldSeparators := numberOfFields - 1

		for _, proposal := range proposals {
			proposalsLength += proposalLength(proposal)
		}

		return objectBraces + fieldSeparators + proposalsLength
	}
}

func proposalLength(proposal ocr2keepers.CoordinatedBlockProposal) int {
	numberOfFields := 3 // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1

	upkeepIDNameAndQuotes := 10
	triggerNameAndQuotes := 9
	workIDNameAndQuotes := 8

	valueSizes := 0

	valueSizes += 2 + byte32Length(proposal.UpkeepID) + len(proposal.UpkeepID) - 1 // 2 for brackets, length of bytes, length-1 for commas
	valueSizes += triggerLength(proposal.Trigger)
	valueSizes += 2 + len(proposal.WorkID)

	return objectBraces + numberOfColons + fieldSeparators + upkeepIDNameAndQuotes + triggerNameAndQuotes + workIDNameAndQuotes + valueSizes
}

func blockHistoryLength(blockHistory ocr2keepers.BlockHistory) int {
	if len(blockHistory) == 0 {
		return 2
	} else {
		blockHistoryLength := 0

		numberOfFields := len(blockHistory) // should not be counted, used to coordinate values, we don't include retry interval

		objectBraces := 2
		fieldSeparators := numberOfFields - 1

		for _, block := range blockHistory {
			blockHistoryLength += blockLength(block)
		}

		return objectBraces + fieldSeparators + blockHistoryLength
	}
}

func blockLength(key ocr2keepers.BlockKey) int {
	numberOfFields := 2 // should not be counted, used to coordinate values

	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1

	numberNameAndQuotes := 8
	hashNameAndQuotes := 6

	valueSizes := 0

	valueSizes += uint64Length(uint64(key.Number))
	valueSizes += 2 + byte32Length(key.Hash) + len(key.Hash) - 1 // 2 for brackets, length of bytes, length-1 for commas

	return objectBraces + numberOfColons + fieldSeparators + numberNameAndQuotes + hashNameAndQuotes + valueSizes
}

func DecodeAutomationObservation(data []byte, utg types.UpkeepTypeGetter, wg types.WorkIDGenerator) (AutomationObservation, error) {
	ao := AutomationObservation{}
	err := json.Unmarshal(data, &ao)
	if err != nil {
		return AutomationObservation{}, err
	}
	err = validateAutomationObservation(ao, utg, wg)
	if err != nil {
		return AutomationObservation{}, err
	}
	return ao, nil
}

func validateAutomationObservation(o AutomationObservation, utg types.UpkeepTypeGetter, wg types.WorkIDGenerator) error {
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

	seenPerformables := make(map[string]bool)
	for _, res := range o.Performable {
		if err := validateCheckResult(res, utg, wg); err != nil {
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
	conditionalProposalCount := 0
	logProposalCount := 0
	seenProposals := make(map[string]bool)
	for _, proposal := range o.UpkeepProposals {
		if err := validateUpkeepProposal(proposal, utg, wg); err != nil {
			return err
		}
		if seenProposals[proposal.WorkID] {
			return fmt.Errorf("proposals cannot have duplicate workIDs")
		}
		seenProposals[proposal.WorkID] = true
		if utg(proposal.UpkeepID) == types.ConditionTrigger {
			conditionalProposalCount++
		} else if utg(proposal.UpkeepID) == types.LogTrigger {
			logProposalCount++
		}
	}
	if conditionalProposalCount > ObservationConditionalsProposalsLimit {
		return fmt.Errorf("conditional upkeep proposals length cannot be greater than %d", ObservationConditionalsProposalsLimit)
	}
	if logProposalCount > ObservationLogRecoveryProposalsLimit {
		return fmt.Errorf("log upkeep proposals length cannot be greater than %d", ObservationLogRecoveryProposalsLimit)
	}

	return nil
}

// Validates the check result fields sent within an observation
func validateCheckResult(r ocr2keepers.CheckResult, utg types.UpkeepTypeGetter, wg types.WorkIDGenerator) error {
	if r.PipelineExecutionState != 0 || r.Retryable {
		return fmt.Errorf("check result cannot have failed execution state")
	}
	if !r.Eligible || r.IneligibilityReason != 0 {
		return fmt.Errorf("check result cannot be ineligible")
	}
	// UpkeepID is contained [32]byte, no validation needed
	if err := validateTriggerExtensionType(r.Trigger, utg(r.UpkeepID)); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}
	if generatedWorkID := wg(r.UpkeepID, r.Trigger); generatedWorkID != r.WorkID {
		return fmt.Errorf("incorrect workID within result")
	}
	if r.GasAllocated == 0 {
		return fmt.Errorf("gas allocated cannot be zero")
	}
	// PerformData is a []byte, no validation needed. Length constraint is handled
	// by maxObservationSize
	if r.FastGasWei == nil {
		return fmt.Errorf("fast gas wei must be present")
	}
	if r.FastGasWei.Cmp(big.NewInt(0)) < 0 || r.FastGasWei.Cmp(uint256Max) > 0 {
		return fmt.Errorf("fast gas wei must be in uint256 range")
	}
	if r.LinkNative == nil {
		return fmt.Errorf("link native must be present")
	}
	if r.LinkNative.Cmp(big.NewInt(0)) < 0 || r.LinkNative.Cmp(uint256Max) > 0 {
		return fmt.Errorf("link native must be in uint256 range")
	}
	return nil
}

func validateUpkeepProposal(p ocr2keepers.CoordinatedBlockProposal, utg types.UpkeepTypeGetter, wg types.WorkIDGenerator) error {
	// No validation is done on Trigger.BlockNumber and Trigger.BlockHash because those
	// get updated with a coordinated quorum block
	ut := utg(p.UpkeepID)
	if err := validateTriggerExtensionType(p.Trigger, ut); err != nil {
		return err
	}
	if generatedWorkID := wg(p.UpkeepID, p.Trigger); generatedWorkID != p.WorkID {
		return fmt.Errorf("incorrect workID within proposal")
	}
	return nil
}

// Validate validates the trigger fields, and any extensions if present.
func validateTriggerExtensionType(t ocr2keepers.Trigger, ut types.UpkeepType) error {
	switch ut {
	case types.ConditionTrigger:
		if t.LogTriggerExtension != nil {
			return fmt.Errorf("log trigger extension cannot be present for condition upkeep")
		}
	case types.LogTrigger:
		if t.LogTriggerExtension == nil {
			return fmt.Errorf("log trigger extension cannot be empty for log upkeep")
		}
	}
	return nil
}
