package plugin

import (
	"bytes"
	"log"
	"testing"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
	"github.com/stretchr/testify/assert"
)

func TestAddBlockHistoryHook_RunHook(t *testing.T) {
	tests := []struct {
		name           string
		existingBlocks types.BlockHistory
		blockHistory   types.BlockHistory
		limit          int
		expectedOutput types.BlockHistory
	}{
		{
			name: "Add block history to observation",
			blockHistory: types.BlockHistory{
				{Number: 1},
				{Number: 2},
				{Number: 3},
			},
			limit:          10,
			expectedOutput: types.BlockHistory{{Number: 1}, {Number: 2}, {Number: 3}},
		},
		{
			name:           "Empty block history",
			blockHistory:   types.BlockHistory{},
			limit:          10,
			expectedOutput: types.BlockHistory{},
		},
		{
			name: "Overwrites existing block history",
			existingBlocks: types.BlockHistory{
				{Number: 1},
				{Number: 2},
				{Number: 3},
			},
			blockHistory:   types.BlockHistory{},
			limit:          10,
			expectedOutput: types.BlockHistory{},
		},
		{
			name:           "limits blocks added",
			existingBlocks: types.BlockHistory{},
			blockHistory: types.BlockHistory{
				{Number: 1},
				{Number: 2},
				{Number: 3},
			},
			limit: 2,
			expectedOutput: types.BlockHistory{
				{Number: 1},
				{Number: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mock MetadataStore
			mockMetadataStore := &mocks.MockMetadataStore{}
			mockMetadataStore.On("GetBlockHistory").Return(tt.blockHistory)

			// Prepare logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)

			// Create the hook with mock MetadataStore and logger
			addBlockHistoryHook := NewAddBlockHistoryHook(mockMetadataStore, logger)

			// Prepare automation observation
			obs := &ocr2keepersv3.AutomationObservation{
				BlockHistory: tt.existingBlocks,
			}

			// Run the hook
			err := addBlockHistoryHook.RunHook(obs, tt.limit)

			// Assert that the hook function executed without error
			assert.NoError(t, err)

			// Assert that the observation's BlockHistory matches the expected output
			assert.Equal(t, tt.expectedOutput, obs.BlockHistory)
		})
	}
}
