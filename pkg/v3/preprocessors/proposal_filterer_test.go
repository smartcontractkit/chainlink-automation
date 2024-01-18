package preprocessors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestProposalFilterer_PreProcess(t *testing.T) {
	metadata := &mockMetadataStore{
		ViewProposalsFn: func(utype types.UpkeepType) []commontypes.CoordinatedBlockProposal {
			return []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID2",
				},
			}
		},
	}
	filterer := &proposalFilterer{
		metadata:   metadata,
		upkeepType: types.LogTrigger,
	}
	payloads, err := filterer.PreProcess(context.Background(), []commontypes.UpkeepPayload{
		{
			WorkID: "workID1",
		},
		{
			WorkID: "workID2",
		},
		{
			WorkID: "workID3",
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, []commontypes.UpkeepPayload{{WorkID: "workID1"}, {WorkID: "workID3"}}, payloads)
}

type mockMetadataStore struct {
	types.MetadataStore
	ViewProposalsFn func(utype types.UpkeepType) []commontypes.CoordinatedBlockProposal
}

func (s *mockMetadataStore) ViewProposals(utype types.UpkeepType) []commontypes.CoordinatedBlockProposal {
	return s.ViewProposalsFn(utype)
}
