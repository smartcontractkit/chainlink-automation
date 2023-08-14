package ocr2keepers

import (
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
