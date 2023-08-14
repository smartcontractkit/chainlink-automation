package ocr2keepers

import (
	"math/big"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/stretchr/testify/assert"
)

var conditionalUpkeepID = [32]byte{1}
var logUpkeepID = [32]byte{2}
var conditionalTrigger = types.Trigger{
	BlockNumber: 10,
	BlockHash:   [32]byte{1},
}
var logTrigger = types.Trigger{
	BlockNumber: 10,
	BlockHash:   [32]byte{1},
	LogTriggerExtension: &types.LogTriggerExtension{
		TxHash:      [32]byte{1},
		Index:       0,
		BlockHash:   [32]byte{1},
		BlockNumber: 5,
	},
}
var validConditionalResult = types.CheckResult{
	PipelineExecutionState: 0,
	Retryable:              false,
	Eligible:               true,
	IneligibilityReason:    0,
	UpkeepID:               conditionalUpkeepID,
	Trigger:                conditionalTrigger,
	WorkID:                 mockWorkIDGenerator(conditionalUpkeepID, conditionalTrigger),
	GasAllocated:           100,
	PerformData:            []byte("testing"),
	FastGasWei:             big.NewInt(100),
	LinkNative:             big.NewInt(100),
}
var validLogResult = types.CheckResult{
	PipelineExecutionState: 0,
	Retryable:              false,
	Eligible:               true,
	IneligibilityReason:    0,
	UpkeepID:               logUpkeepID,
	Trigger:                logTrigger,
	WorkID:                 mockWorkIDGenerator(logUpkeepID, logTrigger),
	GasAllocated:           100,
	PerformData:            []byte("testing"),
	FastGasWei:             big.NewInt(100),
	LinkNative:             big.NewInt(100),
}
var validConditionalProposal = types.CoordinatedBlockProposal{
	UpkeepID: conditionalUpkeepID,
	Trigger:  conditionalTrigger,
	WorkID:   mockWorkIDGenerator(conditionalUpkeepID, conditionalTrigger),
}
var validLogProposal = types.CoordinatedBlockProposal{
	UpkeepID: logUpkeepID,
	Trigger:  logTrigger,
	WorkID:   mockWorkIDGenerator(logUpkeepID, logTrigger),
}
var validBlockHistory = types.BlockHistory{
	{
		Number: 10,
		Hash:   [32]byte{1},
	},
}

func TestValidAutomationObservation(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	decoded, err := DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation observation")

	assert.Equal(t, ao, decoded, "final result from encoding and decoding should match")
}

func mockUpkeepTypeGetter(id types.UpkeepIdentifier) types.UpkeepType {
	if id == conditionalUpkeepID {
		return types.ConditionTrigger
	}
	return types.LogTrigger
}

func mockWorkIDGenerator(id types.UpkeepIdentifier, trigger types.Trigger) string {
	wid := string(id[:])
	if trigger.LogTriggerExtension != nil {
		wid += string(trigger.LogTriggerExtension.LogIdentifier())
	}
	return wid
}
