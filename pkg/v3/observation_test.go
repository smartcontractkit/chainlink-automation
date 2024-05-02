package ocr2keepers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

var conditionalUpkeepID = [32]byte{1}
var logUpkeepID = [32]byte{2}
var conditionalTrigger = commontypes.Trigger{
	BlockNumber: 10,
	BlockHash:   [32]byte{1},
}
var logTrigger = commontypes.Trigger{
	BlockNumber: 10,
	BlockHash:   [32]byte{1},
	LogTriggerExtension: &commontypes.LogTriggerExtension{
		TxHash:      [32]byte{1},
		Index:       0,
		BlockHash:   [32]byte{1},
		BlockNumber: 5,
	},
}
var validConditionalResult = commontypes.CheckResult{
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
var validLogResult = commontypes.CheckResult{
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
var validConditionalProposal = commontypes.CoordinatedBlockProposal{
	UpkeepID: conditionalUpkeepID,
	Trigger:  conditionalTrigger,
	WorkID:   mockWorkIDGenerator(conditionalUpkeepID, conditionalTrigger),
}
var validLogProposal = commontypes.CoordinatedBlockProposal{
	UpkeepID: logUpkeepID,
	Trigger:  logTrigger,
	WorkID:   mockWorkIDGenerator(logUpkeepID, logTrigger),
}
var validBlockHistory = commontypes.BlockHistory{
	{
		Number: 10,
		Hash:   [32]byte{1},
	},
}
var validObservation = AutomationObservation{
	Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
	UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
	BlockHistory:    validBlockHistory,
}
var expectedEncodedObservation []byte

func init() {
	b, err := os.ReadFile("fixtures/expected_encoded_observation.txt")
	if err != nil {
		panic(err)
	}
	expectedEncodedObservation, err = hex.DecodeString(string(b))
	if err != nil {
		panic(err)
	}
}

func TestValidAutomationObservation(t *testing.T) {
	encoded, err := validObservation.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	decoded, err := DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation observation")

	assert.Equal(t, validObservation, decoded, "final result from encoding and decoding should match")
}

func TestAutomationObservationEncodeBackwardsCompatibility(t *testing.T) {
	encoded, err := validObservation.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	if !bytes.Equal(encoded, expectedEncodedObservation) {
		assert.Fail(t,
			"encoded observation does not match expected encoded observation; "+
				"this means a breaking change has been made to the observation encoding function; "+
				"only update this test if non-backwards-compatible changes are necessary",
		)
	}
}

func TestLargeBlockHistory(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    commontypes.BlockHistory{},
	}
	for i := 0; i < ObservationBlockHistoryLimit+1; i++ {
		ao.BlockHistory = append(ao.BlockHistory, commontypes.BlockKey{
			Number: commontypes.BlockNumber(i + 1),
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
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    commontypes.BlockHistory{},
	}
	for i := 0; i < 2; i++ {
		ao.BlockHistory = append(ao.BlockHistory, commontypes.BlockKey{
			Number: commontypes.BlockNumber(1),
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
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationPerformablesLimit+1; i++ {
		newConditionalResult := validConditionalResult
		uid := commontypes.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newConditionalResult.UpkeepID = uid
		newConditionalResult.WorkID = mockWorkIDGenerator(newConditionalResult.UpkeepID, newConditionalResult.Trigger)
		ao.Performable = append(ao.Performable, newConditionalResult)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "performable length cannot be greater than")
}

func TestDuplicatePerformable(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
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
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationConditionalsProposalsLimit+ObservationLogRecoveryProposalsLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := commontypes.UpkeepIdentifier{}
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
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationConditionalsProposalsLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := commontypes.UpkeepIdentifier{}
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
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	for i := 0; i < ObservationLogRecoveryProposalsLimit+1; i++ {
		newProposal := validLogProposal
		uid := commontypes.UpkeepIdentifier{}
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
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
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

func TestInvalidPipelineExecutionState(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validConditionalResult
	invalidPerformable.PipelineExecutionState = 1
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "check result cannot have failed execution state")
}

func TestInvalidRetryable(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validConditionalResult
	invalidPerformable.Retryable = true
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "check result cannot have failed execution state")
}

func TestInvalidEligibility(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validConditionalResult
	invalidPerformable.Eligible = false
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "check result cannot be ineligible")
}

func TestInvalidIneligibilityReason(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validConditionalResult
	invalidPerformable.IneligibilityReason = 1
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "check result cannot be ineligible")
}

func TestInvalidTriggerTypeConditional(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validConditionalResult
	invalidPerformable.Trigger = logTrigger
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid trigger")
}

func TestInvalidTriggerTypeLog(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.Trigger = conditionalTrigger
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid trigger")
}

func TestInvalidWorkID(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.WorkID = "invalid"
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "incorrect workID within result")
}

func TestInvalidGasAllocated(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.GasAllocated = 0
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "gas allocated cannot be zero")
}

func TestNilFastGas(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.FastGasWei = nil
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "fast gas wei must be present")
}

func TestInvalidFastGasNegative(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.FastGasWei = big.NewInt(-1)
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "fast gas wei must be in uint256 range")
}

func TestInvalidFastGasTooBig(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.FastGasWei, _ = big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639936", 10)
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "fast gas wei must be in uint256 range")
}

func TestNilLinkNative(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.LinkNative = nil
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "link native must be present")
}

func TestInvalidLinkNativeNegative(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.LinkNative = big.NewInt(-1)
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "link native must be in uint256 range")
}

func TestInvalidLinkNativeTooBig(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{validConditionalProposal, validLogProposal},
		BlockHistory:    validBlockHistory,
	}
	invalidPerformable := validLogResult
	invalidPerformable.LinkNative, _ = big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639936", 10)
	ao.Performable = append(ao.Performable, invalidPerformable)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "link native must be in uint256 range")
}

func TestInvalidWorkIDProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	invalidProposal := validLogProposal
	invalidProposal.WorkID = "invalid"
	ao.UpkeepProposals = append(ao.UpkeepProposals, invalidProposal)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "incorrect workID within proposal")
}

func TestInvalidConditionalProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	invalidProposal := validConditionalProposal
	invalidProposal.Trigger = logTrigger
	ao.UpkeepProposals = append(ao.UpkeepProposals, invalidProposal)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "log trigger extension cannot be present for condition upkeep")
}

func TestInvalidLogProposal(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{validConditionalResult, validLogResult},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    validBlockHistory,
	}
	invalidProposal := validLogProposal
	invalidProposal.Trigger = conditionalTrigger
	ao.UpkeepProposals = append(ao.UpkeepProposals, invalidProposal)
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "log trigger extension cannot be empty for log upkeep")
}

func TestLargeObservationSize(t *testing.T) {
	ao := AutomationObservation{
		Performable:     []commontypes.CheckResult{},
		UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
		BlockHistory:    commontypes.BlockHistory{},
	}
	for i := 0; i < ObservationBlockHistoryLimit; i++ {
		ao.BlockHistory = append(ao.BlockHistory, commontypes.BlockKey{
			Number: commontypes.BlockNumber(i + 1),
			Hash:   [32]byte{1},
		})
	}
	largePerformData := [10001]byte{}
	for i := 0; i < ObservationPerformablesLimit; i++ {
		newResult := validLogResult
		uid := commontypes.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 10001)))
		newResult.UpkeepID = uid
		newResult.WorkID = mockWorkIDGenerator(newResult.UpkeepID, newResult.Trigger)
		newResult.PerformData = largePerformData[:]
		ao.Performable = append(ao.Performable, newResult)
	}
	for i := 0; i < ObservationConditionalsProposalsLimit; i++ {
		newProposal := validConditionalProposal
		uid := commontypes.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	for i := 0; i < ObservationLogRecoveryProposalsLimit; i++ {
		newProposal := validLogProposal
		uid := commontypes.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1001)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.UpkeepProposals = append(ao.UpkeepProposals, newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation observation")

	decoded, err := DecodeAutomationObservation(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation observation")

	assert.Equal(t, ao, decoded, "final result from encoding and decoding should match")
	assert.Less(t, len(encoded), MaxObservationLength, "encoded observation should be less than maxObservationSize")
}

func TestObservationLength(t *testing.T) {
	for _, tc := range []struct {
		name         string
		observation  AutomationObservation
		expectedJSON string
		expectedSize int
	}{
		{
			name:         "Empty observation has 63 bytes of JSON",
			observation:  AutomationObservation{},
			expectedJSON: `{"Performable":null,"UpkeepProposals":null,"BlockHistory":null}`,
			expectedSize: 63,
		},
		{
			name: "With non-nil performables, 61 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{},
			},
			expectedJSON: `{"Performable":[],"UpkeepProposals":null,"BlockHistory":null}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil upkeep proposals, 61 bytes of JSON",
			observation: AutomationObservation{
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
			},
			expectedJSON: `{"Performable":null,"UpkeepProposals":[],"BlockHistory":null}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil block history, 61 bytes of JSON",
			observation: AutomationObservation{
				BlockHistory: commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":null,"UpkeepProposals":null,"BlockHistory":[]}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil performable, upkeep proposals and block history, 57 bytes of JSON",
			observation: AutomationObservation{
				Performable:     []commontypes.CheckResult{},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 57,
		},
		{
			name: "With one empty performable, upkeep proposals and block history, 438 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 438,
		},
		{
			name: "With two empty performables, empty upkeep proposals and block history, 820 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{},
					{},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 820,
		},
		{
			name: "With one partially populated performable, empty upkeep proposals and block history, 473 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":null},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 473,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and block history, 684 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{1, 2, 3, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"AQIDBA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 684,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and block history, 692 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 692,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and non empty block history, 955 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory: commontypes.BlockHistory{
					commontypes.BlockKey{
						Number: 1,
						Hash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 2,
						Hash:   [32]byte{2, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 3,
						Hash:   [32]byte{3, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[{"Number":1,"Hash":[1,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":2,"Hash":[2,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":3,"Hash":[3,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]}]}`,
			expectedSize: 955,
		},
		{
			name: "With one fully populated performable, upkeep proposals and block history, 2283 bytes of JSON",
			observation: AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID: "WorkID1",
					},
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 202003244343430,
							BlockHash:   [32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{22, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 202003244343430,
							},
						},
						WorkID: "WorkID2",
					},
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 302003244343430,
							BlockHash:   [32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{33, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 302003244343430,
							},
						},
						WorkID: "WorkID3",
					},
				},
				BlockHistory: commontypes.BlockHistory{
					commontypes.BlockKey{
						Number: 1,
						Hash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 2,
						Hash:   [32]byte{2, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 3,
						Hash:   [32]byte{3, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[{"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"WorkID1"},{"UpkeepID":[22,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":202003244343430,"BlockHash":[22,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[22,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[22,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":202003244343430}},"WorkID":"WorkID2"},{"UpkeepID":[33,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":302003244343430,"BlockHash":[33,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[33,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[33,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":302003244343430}},"WorkID":"WorkID3"}],"BlockHistory":[{"Number":1,"Hash":[1,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":2,"Hash":[2,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":3,"Hash":[3,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]}]}`,
			expectedSize: 2283,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.observation)
			assert.NoError(t, err)
			assert.Equal(t, len(b), tc.observation.Length())
			assert.Equal(t, tc.expectedJSON, string(b))
			assert.Equal(t, tc.expectedSize, len(b))
		})

	}
}

func mockUpkeepTypeGetter(id commontypes.UpkeepIdentifier) types.UpkeepType {
	if id == conditionalUpkeepID {
		return types.ConditionTrigger
	}
	if id.BigInt().Cmp(big.NewInt(1000)) < 0 {
		return types.ConditionTrigger
	}
	return types.LogTrigger
}

func mockWorkIDGenerator(id commontypes.UpkeepIdentifier, trigger commontypes.Trigger) string {
	wid := id.String()
	if trigger.LogTriggerExtension != nil {
		wid += string(trigger.LogTriggerExtension.LogIdentifier())
	}
	return wid
}
