package hooks

import (
	"bytes"
	"log"
	"testing"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddFromStagingHook_RunHook(t *testing.T) {
	tests := []struct {
		name                     string
		initialObservation       ocr2keepersv3.AutomationObservation
		resultStoreResults       []types.CheckResult
		resultStoreErr           error
		coordinatorFilterResults []types.CheckResult
		coordinatorErr           error
		limit                    int
		observationPerformable   int
		expectedErr              error
		expectedLogMsg           string
	}{
		{
			name:               "Add results to observation",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
				{UpkeepID: [32]byte{3}},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
				{UpkeepID: [32]byte{3}},
			},
			observationPerformable: 3,
			expectedLogMsg:         "adding 3 results to observation",
		},
		{
			name:                     "Empty result store",
			initialObservation:       ocr2keepersv3.AutomationObservation{},
			resultStoreResults:       []types.CheckResult{},
			coordinatorFilterResults: []types.CheckResult{},
			limit:                    10,
			observationPerformable:   0,
			expectedLogMsg:           "adding 0 results to observation",
		},
		{
			name:               "Filtered coordinator results observation",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
				{UpkeepID: [32]byte{3}},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
			},
			observationPerformable: 2,
			expectedLogMsg:         "adding 2 results to observation",
		},
		{
			name: "Existing results in observation appended",
			initialObservation: ocr2keepersv3.AutomationObservation{
				Performable: []types.CheckResult{{UpkeepID: [32]byte{3}}},
			},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
			},
			observationPerformable: 3,
			expectedLogMsg:         "adding 2 results to observation",
		},
		{
			name:               "limits applied",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
			},
			limit: 1,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}},
				{UpkeepID: [32]byte{2}},
			},
			observationPerformable: 1,
			expectedLogMsg:         "adding 1 results to observation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mock result store
			mockResultStore := &mocks.MockResultStore{}
			mockResultStore.On("View").Return(tt.resultStoreResults, tt.resultStoreErr)

			// Prepare mock coordinator
			mockCoordinator := &mocks.MockCoordinator{}
			mockCoordinator.On("FilterResults", mock.Anything).Return(tt.coordinatorFilterResults, tt.coordinatorErr)

			// Prepare logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)

			// Prepare observation and random source
			obs := &tt.initialObservation

			// Create the hook with mock result store, coordinator, and logger
			addFromStagingHook := NewAddFromStagingHook(mockResultStore, mockCoordinator, logger)

			// Run the hook
			err := addFromStagingHook.RunHook(obs, tt.limit, [16]byte{})

			if tt.expectedErr != nil {
				// Assert that the hook function returns the expected error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				// Assert that the hook function executed without error
				assert.NoError(t, err)

				// Assert that the observation is updated as expected
				assert.Len(t, obs.Performable, tt.observationPerformable)

				// Assert log messages if needed
				assert.Contains(t, logBuf.String(), tt.expectedLogMsg)
			}
		})
	}
}
