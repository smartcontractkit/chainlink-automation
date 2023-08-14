package plugin

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func Test_newCoordinatedBlockProposals_add(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		observations         []ocr2keepers.AutomationObservation
		quorumBlockthreshold int

		wantAllNewProposals []types.CoordinatedBlockProposal
		wantRecentBlocks    map[types.BlockKey]int
		wantQuorum          bool
		wantQuorumBlock     types.BlockKey
	}{
		{
			name: "from a single observation, upkeep proposals are added to the proposals and unique recent blocks are counted",
			observations: []ocr2keepers.AutomationObservation{
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
					},
				},
			},
			quorumBlockthreshold: 0,
			wantAllNewProposals: []types.CoordinatedBlockProposal{
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID1",
				},
			},
			wantRecentBlocks: map[types.BlockKey]int{
				{Number: 3, Hash: [32]byte{3}}: 1,
			},
			wantQuorumBlock: types.BlockKey{
				Number: 3,
				Hash:   [32]byte{3},
			},
			wantQuorum: true,
		},
		{
			name: "from multiple observations, upkeep proposals are added to the proposals and unique recent blocks are counted",
			observations: []ocr2keepers.AutomationObservation{
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
					},
				},
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{2},
							Trigger: types.Trigger{
								BlockNumber: 2,
							},
							WorkID: "workID2",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
					},
				},
			},
			quorumBlockthreshold: 1,
			wantAllNewProposals: []types.CoordinatedBlockProposal{
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID1",
				},
				{
					UpkeepID: [32]byte{2},
					Trigger: types.Trigger{
						BlockNumber: 2,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID2",
				},
			},
			wantRecentBlocks: map[types.BlockKey]int{
				{Number: 3, Hash: [32]byte{3}}: 2,
			},
			wantQuorumBlock: types.BlockKey{
				Number: 3,
				Hash:   [32]byte{3},
			},
			wantQuorum: true,
		},
		{
			name: "upkeep proposals are added to the proposals, including duplicates, and unique recent blocks are counted",
			observations: []ocr2keepers.AutomationObservation{
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
						{
							UpkeepID: [32]byte{2},
							Trigger: types.Trigger{
								BlockNumber: 2,
							},
							WorkID: "workID2",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 2,
							Hash:   [32]byte{2},
						},
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
					},
				},
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
						{
							UpkeepID: [32]byte{2},
							Trigger: types.Trigger{
								BlockNumber: 2,
							},
							WorkID: "workID2",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 2,
							Hash:   [32]byte{2},
						},
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
						{
							Number: 4,
							Hash:   [32]byte{4},
						},
					},
				},
			},
			quorumBlockthreshold: 1,
			wantAllNewProposals: []types.CoordinatedBlockProposal{
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID1",
				},
				{
					UpkeepID: [32]byte{2},
					Trigger: types.Trigger{
						BlockNumber: 2,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID2",
				},
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID1",
				},
				{
					UpkeepID: [32]byte{2},
					Trigger: types.Trigger{
						BlockNumber: 2,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID2",
				},
			},
			wantRecentBlocks: map[types.BlockKey]int{
				{Number: 2, Hash: [32]byte{2}}: 2,
				{Number: 3, Hash: [32]byte{3}}: 2,
				{Number: 4, Hash: [32]byte{4}}: 1,
			},
			wantQuorumBlock: types.BlockKey{
				Number: 3,
				Hash:   [32]byte{3},
			},
			wantQuorum: true,
		},
		{
			name: "too few blocks have been counted, so a quorum block cannot be fetched",
			observations: []ocr2keepers.AutomationObservation{
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
							Hash:   [32]byte{3},
						},
					},
				},
			},
			quorumBlockthreshold: 1,
			wantAllNewProposals: []types.CoordinatedBlockProposal{
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
						BlockHash:   [32]byte{},
					},
					WorkID: "workID1",
				},
			},
			wantRecentBlocks: map[types.BlockKey]int{
				{Number: 3, Hash: [32]byte{3}}: 1,
			},
			wantQuorum: false,
		},
		{
			name: "if a block key that meets the block quorum threshold has a zero hash, a quorum block key cannot be fetched",
			observations: []ocr2keepers.AutomationObservation{
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
						},
					},
				},
				{
					UpkeepProposals: []types.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: types.Trigger{
								BlockNumber: 1,
							},
							WorkID: "workID1",
						},
					},
					BlockHistory: []types.BlockKey{
						{
							Number: 3,
						},
					},
				},
			},
			quorumBlockthreshold: 1,
			wantAllNewProposals: []types.CoordinatedBlockProposal{
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
					},
					WorkID: "workID1",
				},
				{
					UpkeepID: [32]byte{1},
					Trigger: types.Trigger{
						BlockNumber: 1,
					},
					WorkID: "workID1",
				},
			},
			wantRecentBlocks: map[types.BlockKey]int{
				{Number: 3}: 2,
			},
			wantQuorum: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proposals := newCoordinatedBlockProposals(tc.quorumBlockthreshold, 2, 3, [16]byte{1}, log.New(io.Discard, "", 1))
			for _, ao := range tc.observations {
				proposals.add(ao)
			}
			assert.Equal(t, tc.wantAllNewProposals, proposals.allNewProposals)
			assert.Equal(t, tc.wantRecentBlocks, proposals.recentBlocks)
			quorumBlock, ok := proposals.getLatestQuorumBlock()
			if tc.wantQuorum {
				assert.True(t, ok)
				assert.Equal(t, tc.wantQuorumBlock, quorumBlock)
			} else {
				assert.False(t, ok)
			}
		})
	}
}

func Test_proposalExists(t *testing.T) {
	for _, tc := range []struct {
		name          string
		proposals     [][]types.CoordinatedBlockProposal
		checkProposal types.CoordinatedBlockProposal
		exists        bool
	}{
		{
			name: "given zero proposals, the check proposal does not exist",
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID1",
			},
			exists: false,
		},
		{
			name: "given a list of proposals with different workIDs, the check proposal does not exist",
			proposals: [][]types.CoordinatedBlockProposal{
				{
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID2",
					},
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID3",
					},
				},
				{
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID4",
					},
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID5",
					},
				},
			},
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID1",
			},
			exists: false,
		},
		{
			name: "given a list of proposals with different workIDs, the check proposal does exist",
			proposals: [][]types.CoordinatedBlockProposal{
				{
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID2",
					},
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID3",
					},
				},
				{
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID4",
					},
					{
						UpkeepID: types.UpkeepIdentifier([32]byte{1}),
						WorkID:   "workID5",
					},
				},
			},
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID5",
			},
			exists: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.exists, proposalExists(tc.proposals, tc.checkProposal))
		})
	}
}

func Test_performableExists(t *testing.T) {
	for _, tc := range []struct {
		name          string
		proposals     []types.CheckResult
		checkProposal types.CoordinatedBlockProposal
		exists        bool
	}{
		{
			name: "given zero proposals, the check proposal does not exist",
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID1",
			},
			exists: false,
		},
		{
			name: "given a list of proposals with different workIDs, the check proposal does not exist",
			proposals: []types.CheckResult{
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID2",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID3",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID4",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID5",
				},
			},
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID1",
			},
			exists: false,
		},
		{
			name: "given a list of proposals with different workIDs, the check proposal does exist",
			proposals: []types.CheckResult{
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID2",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID3",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID4",
				},
				{
					UpkeepID: types.UpkeepIdentifier([32]byte{1}),
					WorkID:   "workID5",
				},
			},
			checkProposal: types.CoordinatedBlockProposal{
				UpkeepID: types.UpkeepIdentifier([32]byte{1}),
				WorkID:   "workID5",
			},
			exists: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.exists, performableExists(tc.proposals, tc.checkProposal))
		})
	}
}
