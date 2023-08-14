package ocr2keepers

import (
	"math/big"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/stretchr/testify/assert"
)

func TestValidAutomationOutcome(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{validConditionalResult, validLogResult},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{validConditionalProposal, validLogProposal}},
	}
	encoded, err := ao.Encode()
	assert.NoError(t, err, "no error in encoding valid automation outcome")

	decoded, err := DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.NoError(t, err, "no error in decoding valid automation outcome")

	assert.Equal(t, ao, decoded, "final result from encoding and decoding should match")
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
	assert.NoError(t, err, "no error in encoding valid automation observation")

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
	assert.NoError(t, err, "no error in encoding valid automation observation")

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
	assert.NoError(t, err, "no error in encoding valid automation observation")

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
	assert.NoError(t, err, "no error in encoding valid automation observation")

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
	assert.NoError(t, err, "no error in encoding valid automation observation")

	_, err = DecodeAutomationOutcome(encoded, mockUpkeepTypeGetter, mockWorkIDGenerator)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "proposals cannot have duplicate workIDs")
}

// TODO: fix and add
/*
func TestLargeOutcomeSize(t *testing.T) {
	ao := AutomationOutcome{
		AgreedPerformables: []types.CheckResult{},
		SurfacedProposals:  [][]types.CoordinatedBlockProposal{{}},
	}
	largePerformData := [5001]byte{}
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
			uid.FromBigInt(big.NewInt(int64(i + 1001)))
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
	// TODO: fix import cycle. Should be plugin.MaxOutcomeSize
	assert.Less(t, len(encoded), 1000000, "encoded observation should be less than maxObservationSize")
}
*/
