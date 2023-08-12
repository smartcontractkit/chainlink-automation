package flows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestProposalFilterer_PreProcess(t *testing.T) {
	metadata := &mockMetadataStore{
		ViewProposalsFn: func(utype types.UpkeepType) []types.CoordinatedBlockProposal {
			return []types.CoordinatedBlockProposal{
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
	payloads, err := filterer.PreProcess(context.Background(), []types.UpkeepPayload{
		{
			WorkID: "workID1",
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, []types.UpkeepPayload{{WorkID: "workID1"}}, payloads)
}

type mockMetadataStore struct {
	types.MetadataStore
	ViewProposalsFn func(utype types.UpkeepType) []types.CoordinatedBlockProposal
}

func (s *mockMetadataStore) ViewProposals(utype types.UpkeepType) []types.CoordinatedBlockProposal {
	return s.ViewProposalsFn(utype)
}
