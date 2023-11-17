package hooks

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/mock"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
)

func TestRemoveFromMetadataHook_RunHook(t *testing.T) {
	var uid1 types.UpkeepIdentifier = [32]byte{1}
	var uid2 types.UpkeepIdentifier = [32]byte{2}
	var uid3 types.UpkeepIdentifier = [32]byte{3}
	tests := []struct {
		name              string
		surfacedProposals [][]types.CoordinatedBlockProposal
		upkeepTypeGetter  map[types.UpkeepIdentifier]types.UpkeepType
		expectedRemovals  int
	}{
		{
			name: "Remove proposals from metadata store",
			surfacedProposals: [][]types.CoordinatedBlockProposal{
				{
					{UpkeepID: uid1, WorkID: "1"},
					{UpkeepID: uid2, WorkID: "2"},
				},
				{
					{UpkeepID: uid3, WorkID: "3"},
				},
			},
			upkeepTypeGetter: map[types.UpkeepIdentifier]types.UpkeepType{
				uid1: types.ConditionTrigger,
				uid2: types.LogTrigger,
				uid3: types.ConditionTrigger,
			},
			expectedRemovals: 3,
		},
		{
			name: "No proposals to remove",
			surfacedProposals: [][]types.CoordinatedBlockProposal{
				{},
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mock MetadataStore
			mockMetadataStore := &mocks.MockMetadataStore{}
			if tt.expectedRemovals > 0 {
				mockMetadataStore.On("RemoveProposals", mock.Anything).Times(tt.expectedRemovals)
			}

			// Prepare logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)

			// Create the hook with mock MetadataStore, mock UpkeepTypeGetter, and logger
			removeFromMetadataHook := NewRemoveFromMetadataHook(mockMetadataStore, logger)

			// Prepare automation outcome with agreed proposals
			automationOutcome := ocr2keepersv3.AutomationOutcome{
				SurfacedProposals: tt.surfacedProposals,
			}
			// Run the hook
			removeFromMetadataHook.RunHook(automationOutcome)

			mockMetadataStore.AssertExpectations(t)
		})
	}
}
