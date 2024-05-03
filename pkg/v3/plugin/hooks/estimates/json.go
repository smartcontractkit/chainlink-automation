package estimates

import (
	"encoding/hex"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"strconv"
)

func ObservationLength(observation *ocr2keepers.AutomationObservation) int {
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

func performablesLength(results []automation.CheckResult) int {
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

func performableLength(result automation.CheckResult) int {
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

func triggerLength(t automation.Trigger) int {
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

func logTriggerExtensionLength(extension *automation.LogTriggerExtension) int {
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

func upkeepProposalsLength(proposals []automation.CoordinatedBlockProposal) int {
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

func proposalLength(proposal automation.CoordinatedBlockProposal) int {
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

func blockHistoryLength(blockHistory automation.BlockHistory) int {
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

func blockLength(key automation.BlockKey) int {
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
