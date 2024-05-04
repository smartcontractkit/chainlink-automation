package estimates

import (
	"encoding/hex"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"strconv"
)

const (
	nullFieldLength   = 4
	emptyObjectLength = 2
)

func ObservationLength(observation *ocr2keepers.AutomationObservation) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(3)

	performablesWithQuotes := 13
	upkeepProposalsWithQuotes := 17
	blockHistoryWithQuotes := 14

	fieldValues := 0
	if observation.Performable == nil {
		fieldValues += nullFieldLength
	} else {
		fieldValues += checkResultsLength(observation.Performable)
	}

	if observation.UpkeepProposals == nil {
		fieldValues += nullFieldLength
	} else {
		fieldValues += coordinatedBlockProposalsLength(observation.UpkeepProposals)
	}

	if observation.BlockHistory == nil {
		fieldValues += nullFieldLength
	} else {
		fieldValues += blockHistoryLength(observation.BlockHistory)
	}

	return objectBraces + fieldSeparators + numberOfColons + performablesWithQuotes +
		upkeepProposalsWithQuotes + blockHistoryWithQuotes + fieldValues
}

func checkResultsLength(results []automation.CheckResult) int {
	if len(results) == 0 {
		return emptyObjectLength
	} else {
		objectBraces, _, fieldSeparators := countDelimitersForFields(len(results))

		performablesLength := 0

		for _, result := range results {
			performablesLength += checkResultLength(result)
		}

		return objectBraces + fieldSeparators + performablesLength
	}
}

func checkResultLength(result automation.CheckResult) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(11)

	pipelineExecutionStateWithQuotes := 24
	retryableWithQuotes := 11
	eligibleWithQuotes := 10
	ineligibilityReasonWithQuotes := 21
	upkeepIDWithQuotes := 10
	triggerWithQuotes := 9
	workIDWithQuotes := 8
	gasAllocatedWithQuotes := 14
	performDataWithQuotes := 13
	fastGasWeiWithQuotes := 12
	linkNativeWithQuotes := 12

	valueSizes := 0
	valueSizes += uint8Length(result.PipelineExecutionState)
	valueSizes += boolLength(result.Retryable)
	valueSizes += boolLength(result.Eligible)
	valueSizes += uint8Length(result.IneligibilityReason)
	valueSizes += byte32Length(result.UpkeepID)
	valueSizes += triggerLength(result.Trigger)
	valueSizes += stringLength(result.WorkID)
	valueSizes += uint64Length(result.GasAllocated)

	if result.PerformData == nil {
		valueSizes += nullFieldLength
	} else {
		valueSizes += stringLength(hex.EncodeToString(result.PerformData))
	}

	if result.FastGasWei == nil {
		valueSizes += nullFieldLength
	} else {
		valueSizes += len(result.FastGasWei.String()) // quotes aren't included in big int json
	}

	if result.LinkNative == nil {
		valueSizes += nullFieldLength
	} else {
		valueSizes += len(result.LinkNative.String()) // quotes aren't included in big int json
	}

	return objectBraces + numberOfColons + fieldSeparators + pipelineExecutionStateWithQuotes + retryableWithQuotes +
		eligibleWithQuotes + ineligibilityReasonWithQuotes + upkeepIDWithQuotes + triggerWithQuotes + workIDWithQuotes +
		gasAllocatedWithQuotes + performDataWithQuotes + fastGasWeiWithQuotes + linkNativeWithQuotes + valueSizes
}

func coordinatedBlockProposalsLength(proposals []automation.CoordinatedBlockProposal) int {
	if len(proposals) == 0 {
		return emptyObjectLength
	} else {
		objectBraces, _, fieldSeparators := countDelimitersForFields(len(proposals))

		proposalsLength := 0

		for _, proposal := range proposals {
			proposalsLength += coordinatedBlockProposalLength(proposal)
		}

		return objectBraces + fieldSeparators + proposalsLength
	}
}

func coordinatedBlockProposalLength(proposal automation.CoordinatedBlockProposal) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(3)

	upkeepIDWithQuotes := 10
	triggerWithQuotes := 9
	workIDWithQuotes := 8

	valueSizes := 0

	valueSizes += byte32Length(proposal.UpkeepID)
	valueSizes += triggerLength(proposal.Trigger)
	valueSizes += stringLength(proposal.WorkID)

	return objectBraces + numberOfColons + fieldSeparators + upkeepIDWithQuotes +
		triggerWithQuotes + workIDWithQuotes + valueSizes
}

func triggerLength(t automation.Trigger) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(3)

	blockNumberWithQuotes := 13
	blockHashWithQuotes := 11
	logTriggerExtensionWithQuotes := 21

	valueSizes := 0

	valueSizes += uint64Length(uint64(t.BlockNumber))
	valueSizes += byte32Length(t.BlockHash)

	if t.LogTriggerExtension == nil {
		valueSizes += nullFieldLength
	} else {
		valueSizes += logTriggerExtensionLength(t.LogTriggerExtension)
	}

	return objectBraces + numberOfColons + fieldSeparators + blockNumberWithQuotes +
		blockHashWithQuotes + logTriggerExtensionWithQuotes + valueSizes
}

func logTriggerExtensionLength(extension *automation.LogTriggerExtension) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(4)

	txHashWithQuotes := 8
	indexWithQuotes := 7
	blockHashWithQuotes := 11
	blockNumberWithQuotes := 13

	valueSizes := 0

	valueSizes += byte32Length(extension.TxHash)
	valueSizes += uint32Length(extension.Index)
	valueSizes += uint64Length(uint64(extension.BlockNumber))
	valueSizes += byte32Length(extension.BlockHash)

	return objectBraces + numberOfColons + fieldSeparators + txHashWithQuotes +
		indexWithQuotes + blockNumberWithQuotes + blockHashWithQuotes + valueSizes
}

func blockHistoryLength(blockHistory automation.BlockHistory) int {
	if len(blockHistory) == 0 {
		return emptyObjectLength
	} else {
		objectBraces, _, fieldSeparators := countDelimitersForFields(len(blockHistory))

		blockHistoryLength := 0
		for _, block := range blockHistory {
			blockHistoryLength += blockKeyLength(block)
		}

		return objectBraces + fieldSeparators + blockHistoryLength
	}
}

func blockKeyLength(key automation.BlockKey) int {
	objectBraces, numberOfColons, fieldSeparators := countDelimitersForFields(2)

	numberWithQuotes := 8
	hashWithQuotes := 6

	valueSizes := 0

	valueSizes += uint64Length(uint64(key.Number))
	valueSizes += byte32Length(key.Hash)

	return objectBraces + numberOfColons + fieldSeparators + numberWithQuotes + hashWithQuotes + valueSizes
}

func byteSliceLength(b []byte) int {
	length := emptyObjectLength

	for _, x := range b {
		length += uint8Length(x)
	}

	return length + len(b) - 1
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

func stringLength(s string) int {
	return 2 + len(s)
}

func countDelimitersForFields(numberOfFields int) (int, int, int) {
	objectBraces := 2
	numberOfColons := numberOfFields
	fieldSeparators := numberOfFields - 1
	return objectBraces, numberOfColons, fieldSeparators
}
