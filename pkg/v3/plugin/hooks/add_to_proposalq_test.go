package hooks

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestAddToProposalQHook_RunHook(t *testing.T) {
	tests := []struct {
		name              string
		automationOutcome ocr2keepersv3.AutomationOutcome
		expectedQueueSize int
		expectedLog       string
	}{
		{
			name: "Happy path add proposals to queue",
			automationOutcome: ocr2keepersv3.AutomationOutcome{
				SurfacedProposals: [][]types.CoordinatedBlockProposal{
					{{WorkID: "1"}, {WorkID: "2"}},
					{{WorkID: "3"}},
				},
			},
			expectedQueueSize: 3,
			expectedLog:       "Added 3 proposals from outcome",
		},
		{
			name: "Empty automation outcome",
			automationOutcome: ocr2keepersv3.AutomationOutcome{
				SurfacedProposals: [][]types.CoordinatedBlockProposal{},
			},
			expectedQueueSize: 0,
			expectedLog:       "Added 0 proposals from outcome",
		},
		{
			name: "Multiple rounds with proposals",
			automationOutcome: ocr2keepersv3.AutomationOutcome{
				SurfacedProposals: [][]types.CoordinatedBlockProposal{
					{{WorkID: "1"}, {WorkID: "2"}},
					{{WorkID: "3"}},
					{{WorkID: "4"}, {WorkID: "5"}, {WorkID: "6"}},
				},
			},
			expectedQueueSize: 6,
			expectedLog:       "Added 6 proposals from outcome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upkeepTypeGetter := func(uid types.UpkeepIdentifier) types.UpkeepType {
				return types.UpkeepType(uid[15])
			}
			proposalQ := stores.NewProposalQueue(upkeepTypeGetter)

			// Prepare mock logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)

			// Create the hook with the proposal queue and logger
			addToProposalQHook := NewAddToProposalQHook(proposalQ, logger)

			// Run the hook
			err := addToProposalQHook.RunHook(tt.automationOutcome)

			// Assert that the hook function executed without error
			assert.NoError(t, err)

			// Assert that the correct number of proposals were added to the queue
			assert.Equal(t, tt.expectedQueueSize, proposalQ.Size())

			// Assert log messages if needed
			assert.Contains(t, logBuf.String(), tt.expectedLog)
		})
	}
}
