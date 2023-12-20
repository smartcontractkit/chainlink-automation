package hooks

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	types "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation/mocks"
)

func TestAddFromStagingHook_RunHook(t *testing.T) {
	tests := []struct {
		name                     string
		initialObservation       ocr2keepersv3.AutomationObservation
		resultStoreResults       []types.CheckResult
		resultStoreErr           error
		coordinatorFilterResults []types.CheckResult
		coordinatorErr           error
		rSrc                     [16]byte
		limit                    int
		observationWorkIDs       []string
		expectedErr              error
		expectedLogMsg           string
	}{
		{
			name:               "Add results to observation",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			observationWorkIDs: []string{"30a", "20b", "10c"},
			expectedLogMsg:     "adding 3 results to observation",
		},
		{
			name:                     "Empty result store",
			initialObservation:       ocr2keepersv3.AutomationObservation{},
			resultStoreResults:       []types.CheckResult{},
			coordinatorFilterResults: []types.CheckResult{},
			limit:                    10,
			observationWorkIDs:       []string{},
			expectedLogMsg:           "adding 0 results to observation",
		},
		{
			name:               "Filtered coordinator results observation",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			observationWorkIDs: []string{"20b", "10c"},
			expectedLogMsg:     "adding 2 results to observation",
		},
		{
			name: "Existing results in observation appended",
			initialObservation: ocr2keepersv3.AutomationObservation{
				Performable: []types.CheckResult{{UpkeepID: [32]byte{3}, WorkID: "30a"}},
			},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			limit: 10,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			observationWorkIDs: []string{"30a", "20b", "10c"},
			expectedLogMsg:     "adding 2 results to observation",
		},
		{
			name:               "limits applied",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			limit: 2,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			observationWorkIDs: []string{"30a", "20b"},
			expectedLogMsg:     "adding 2 results to observation",
		},
		{
			name:               "limits applied in same order with same rSrc",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			limit: 1,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			observationWorkIDs: []string{"30a"},
			expectedLogMsg:     "adding 1 results to observation",
		},
		{
			name:               "limits applied in different order with different rSrc",
			initialObservation: ocr2keepersv3.AutomationObservation{},
			resultStoreResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			rSrc:  [16]byte{1},
			limit: 2,
			coordinatorFilterResults: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			observationWorkIDs: []string{"10c", "20b"},
			expectedLogMsg:     "adding 2 results to observation",
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
			err := addFromStagingHook.RunHook(obs, tt.limit, tt.rSrc)

			if tt.expectedErr != nil {
				// Assert that the hook function returns the expected error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				// Assert that the hook function executed without error
				assert.NoError(t, err)

				obsW := []string{}
				for _, r := range obs.Performable {
					obsW = append(obsW, r.WorkID)
				}
				// Assert that the observation is updated as expected
				assert.Equal(t, obsW, tt.observationWorkIDs)

				// Assert log messages if needed
				assert.Contains(t, logBuf.String(), tt.expectedLogMsg)
			}
		})
	}
}

func TestAddFromStagingHook_RunHook_Limits(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		limit    int
		expected int
	}{
		{
			name:     "limit is less than results",
			n:        1000,
			limit:    100,
			expected: 100,
		},
		{
			name:     "limit is greater than results",
			n:        100,
			limit:    200,
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResultStore, mockCoordinator := getMocks(tt.n)
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)
			addFromStagingHook := NewAddFromStagingHook(mockResultStore, mockCoordinator, logger)

			rSrc := [16]byte{1, 1, 2, 2, 3, 3, 4, 4}
			obs := &ocr2keepersv3.AutomationObservation{}

			err := addFromStagingHook.RunHook(obs, tt.limit, rSrc)
			assert.NoError(t, err)
			assert.Len(t, obs.Performable, tt.expected)

			// Run the hook again with the same random source
			// and assert that the results are the same
			mockResultStore2, mockCoordinator2 := getMocks(tt.n)
			addFromStagingHook2 := NewAddFromStagingHook(mockResultStore2, mockCoordinator2, logger)

			obs2 := &ocr2keepersv3.AutomationObservation{}
			err2 := addFromStagingHook2.RunHook(obs2, tt.limit, rSrc)
			assert.NoError(t, err2)
			assert.Len(t, obs.Performable, tt.expected)
			assert.Equal(t, obs.Performable, obs2.Performable)
		})
	}
}

func getMocks(n int) (*mocks.MockResultStore, *mocks.MockCoordinator) {
	mockResults := make([]types.CheckResult, n)
	for i := 0; i < n; i++ {
		mockResults[i] = types.CheckResult{UpkeepID: [32]byte{uint8(i)}, WorkID: fmt.Sprintf("10%d", i)}
	}
	mockResultStore := &mocks.MockResultStore{}
	mockResultStore.On("View").Return(mockResults, nil)
	mockCoordinator := &mocks.MockCoordinator{}
	mockCoordinator.On("FilterResults", mock.Anything).Return(mockResults, nil)

	return mockResultStore, mockCoordinator
}
