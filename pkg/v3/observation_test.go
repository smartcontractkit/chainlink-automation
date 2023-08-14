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

func TestLargeBlockHistory(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    types.BlockHistory{},
	}
	for i := 0; i < ObservationBlockHistoryLimit+1; i++ {
		ao.BlockHistory = append(ao.BlockHistory, types.BlockKey{
			Number: types.BlockNumber(i + 1),
			Hash:   [32]byte{1},
		})
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "block history length cannot be greater than")
}

func TestDuplicateBlockHistory(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    types.BlockHistory{},
	}
	for i := 0; i < 2; i++ {
		ao.BlockHistory = append(ao.BlockHistory, types.BlockKey{
			Number: types.BlockNumber(1),
			Hash:   [32]byte{uint8(i)},
		})
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "block history cannot have duplicate block numbers")
}

func TestLargePerformable(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{},
		UpkeepProposals: []types.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationPerformablesLimit+1; i++ {
		newConditionalResult := validConditionalResult
		newConditionalResult.Trigger.BlockNumber = types.BlockNumber(i + 1)
		newConditionalResult.WorkID = mockWorkIDGenerator(newConditionalResult.UpkeepID, newConditionalResult.Trigger)
		ao.Performable = append(ao.Performable, validConditionalResult)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "performable length cannot be greater than")
}

func TestDuplicatePerformable(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{},
		UpkeepProposals: []types.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < 2; i++ {
		ao.Performable = append(ao.Performable, validConditionalResult)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "performable cannot have duplicate workIDs")
}

func TestLargeProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationConditionalsProposalsLimit+ObservationLogRecoveryProposalsLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "upkeep proposals length cannot be greater than")
}

func TestLargeConditionalProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationConditionalsProposalsLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "conditional upkeep proposals length cannot be greater than")
}

func TestLargeLogProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationLogRecoveryProposalsLimit+1; i++ {
		newProposal := validLogProposal
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1001)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "log upkeep proposals length cannot be greater than")
}

func TestDuplicateProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []types.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []types.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < 2; i++ {
		newProposal := validConditionalProposal
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "proposals cannot have duplicate workIDs")
}

func mockUpkeepTypeGetter(id types.UpkeepIdentifier) types.UpkeepType {
	if id == conditionalUpkeepID {
		return types.ConditionTrigger
	}
	if id.BigInt().Cmp(big.NewInt(1000)) < 0 {
		return types.ConditionTrigger
	}
	return types.LogTrigger
}

func mockWorkIDGenerator(id types.UpkeepIdentifier, trigger types.Trigger) string {
	wid := id.String()
	if trigger.LogTriggerExtension != nil {
		wid += string(trigger.LogTriggerExtension.LogIdentifier())
	}
	return wid
}
