package ocr2keepers

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	types "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

var validOutcome = AutomationOutcome{
	AgreedPerformables: []types.CheckResult{validConditionalResult, validLogResult},
	SurfacedProposals:  [][]types.CoordinatedBlockProposal{{validConditionalProposal, validLogProposal}},
}
var expectedEncodedOutcome []byte

func init() {
	b, err := os.ReadFile("fixtures/expected_encoded_outcome.txt")
	if err != nil {
		panic(err)
	}
	expectedEncodedOutcome, err = hex.DecodeString(string(b))
	if err != nil {
		panic(err)
	}
}

func TestValidAutomationOutcome(t *testing.T) {
	encoded, err := validOutcome.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	decoded, err := DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation outcome")

	assert.Equal(t, validOutcome, decoded, "final result from encoding and decoding should match")
}

func TestAutomationOutcomeEncodeBackwardsCompatibility(t *testing.T) {
	encoded, err := validOutcome.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	if !bytes.Equal(encoded, expectedEncodedOutcome) {
		assert.Fail(t,
			"encoded outcome does not match expected encoded outcome; "+
				"this means a breaking change has been made to the outcome encoding function; "+
				"only update this test if non-backwards-compatible changes are necessary",
		)
	}
}

func TestLargeAgreedPerformables(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{validConditionalProposal, validLogProposal}},
	}
	for i := 0; i < OutcomeAgreedPerformablesLimit+1; i++ {
		newConditionalResult := validConditionalResult
		newConditionalResult.Trigger.BlockNumber = types.BlockNumber(i + 1)
		newConditionalResult.WorkID = mockWorkIDGenerator(newConditionalResult.UpkeepID, newConditionalResult.Trigger)
		ao.AgreedPerformables = append(ao.AgreedPerformables, validConditionalResult)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "outcome performable length cannot be greater than")
}

func TestDuplicateAgreedPerformables(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{validConditionalProposal, validLogProposal}},
	}
	for i := 0; i < 2; i++ {
		ao.AgreedPerformables = append(ao.AgreedPerformables, validConditionalResult)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "agreed performable cannot have duplicate workIDs")
}

func TestLargeProposalHistory(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{validConditionalResult, validLogResult},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{},
	}
	for i := 0; i < OutcomeSurfacedProposalsRoundHistoryLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.SurfacedProposals = append(ao.SurfacedProposals, []types.CoordinatedBlockProposal{newProposal})
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "number of rounds for surfaced proposals cannot be greater than")
}

func TestLargeSurfacedProposalInSingleRound(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{validConditionalResult, validLogResult},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{}},
	}
	for i := 0; i < OutcomeSurfacedProposalsLimit+1; i++ {
		newProposal := validConditionalProposal
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 1)))
		newProposal.UpkeepID = uid
		newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
		ao.SurfacedProposals[0] = append(ao.SurfacedProposals[0], newProposal)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "number of surfaced proposals in a round cannot be greater than")
}

func TestDuplicateSurfaced(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{validConditionalResult, validLogResult},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{}},
	}
	for i := 0; i < 2; i++ {
		ao.SurfacedProposals = append(ao.SurfacedProposals, []types.CoordinatedBlockProposal{validConditionalProposal})
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "proposals cannot have duplicate workIDs")
}

func TestLargeOutcomeSize(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{},
	}
	largePerformData := [10001]byte{}
	for i := 0; i < OutcomeAgreedPerformablesLimit; i++ {
		newResult := validLogResult
		uid := types.UpkeepIdentifier{}
		uid.FromBigInt(big.NewInt(int64(i + 10001)))
		newResult.UpkeepID = uid
		newResult.WorkID = mockWorkIDGenerator(newResult.UpkeepID, newResult.Trigger)
		newResult.PerformData = largePerformData[:]
		ao.AgreedPerformables = append(ao.AgreedPerformables, newResult)
	}
	for i := 0; i < OutcomeSurfacedProposalsRoundHistoryLimit; i++ {
		round := []types.CoordinatedBlockProposal{}
		for j := 0; j < OutcomeSurfacedProposalsLimit; j++ {
			newProposal := validLogProposal
			uid := types.UpkeepIdentifier{}
			uid.FromBigInt(big.NewInt(int64(i*OutcomeSurfacedProposalsLimit + j + 1001)))
			newProposal.UpkeepID = uid
			newProposal.WorkID = mockWorkIDGenerator(newProposal.UpkeepID, newProposal.Trigger)
			round = append(round, newProposal)
		}
		ao.SurfacedProposals = append(ao.SurfacedProposals, round)
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	decoded, err := DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation outcome")

	assert.Equal(t, ao, decoded, "final result from encoding and decoding should match")

	assert.Equal(t, ao, decoded, "final result from encoding and decoding should match")
	assert.Less(t, len(encoded), MaxOutcomeLength, "encoded outcome should be less than maxoutcomeSize")
}
