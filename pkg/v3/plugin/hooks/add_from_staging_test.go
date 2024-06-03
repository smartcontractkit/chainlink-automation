package hooks

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
	types "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
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
			name: "Existing results in observation are overwritten",
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
			observationWorkIDs: []string{"20b", "10c"},
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

func TestAddFromStagingHook_stagedResultSorter(t *testing.T) {
	tests := []struct {
		name                string
		cached              []types.CheckResult
		lastRandSrc         [16]byte
		input               []types.CheckResult
		rSrc                [16]byte
		expected            []types.CheckResult
		expectedCache       map[string]string
		expectedLastRandSrc [16]byte
	}{
		{
			name:                "empty results",
			cached:              []types.CheckResult{},
			input:               []types.CheckResult{},
			rSrc:                [16]byte{1},
			expected:            []types.CheckResult{},
			expectedLastRandSrc: [16]byte{1},
		},
		{
			name: "happy path",
			input: []types.CheckResult{
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			rSrc: [16]byte{1},
			expected: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			expectedCache: map[string]string{
				"10c": "1c0",
				"20b": "2b0",
				"30a": "3a0",
			},
			expectedLastRandSrc: [16]byte{1},
		},
		{
			name: "with cached results",
			cached: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			lastRandSrc: [16]byte{1},
			input: []types.CheckResult{
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			rSrc: [16]byte{1},
			expected: []types.CheckResult{
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			expectedCache: map[string]string{
				"10c": "1c0",
				"20b": "2b0",
				"30a": "3a0",
			},
			expectedLastRandSrc: [16]byte{1},
		},
		{
			name: "with cached results of different rand src",
			cached: []types.CheckResult{
				{UpkeepID: [32]byte{1}, WorkID: "10c"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			lastRandSrc: [16]byte{1},
			input: []types.CheckResult{
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
			},
			rSrc: [16]byte{2},
			expected: []types.CheckResult{
				{UpkeepID: [32]byte{2}, WorkID: "20b"},
				{UpkeepID: [32]byte{3}, WorkID: "30a"},
			},
			expectedCache: map[string]string{
				"20b": "02b",
				"30a": "03a",
			},
			expectedLastRandSrc: [16]byte{2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sorter := stagedResultSorter{
				shuffledIDs: make(map[string]string),
			}

			if len(tc.cached) > 0 {
				sorter.shuffledIDs = make(map[string]string)
				for _, r := range tc.cached {
					sorter.shuffledIDs[r.WorkID] = random.ShuffleString(r.WorkID, tc.lastRandSrc)
				}
				sorter.lastRandSrc = tc.lastRandSrc
			}

			results := sorter.orderResults(tc.input, tc.rSrc)
			assert.Equal(t, len(tc.expected), len(results))
			for i := range results {
				assert.Equal(t, tc.expected[i].WorkID, results[i].WorkID)
			}
			sorter.lock.Lock()
			defer sorter.lock.Unlock()
			assert.Equal(t, tc.expectedLastRandSrc, sorter.lastRandSrc)
			assert.Equal(t, len(tc.expectedCache), len(sorter.shuffledIDs))
			for k, v := range tc.expectedCache {
				assert.Equal(t, v, sorter.shuffledIDs[k])
			}
		})
	}
}

func TestAddByEstimate(t *testing.T) {
	var hook AddFromStagingHook

	var blockHistory types.BlockHistory
	for i := 0; i < ocr2keepersv3.ObservationBlockHistoryLimit; i++ {
		blockHistory = append(blockHistory, types.BlockKey{
			Number: types.BlockNumber(i + 1),
			Hash:   [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		})
	}

	bigNumber := big.NewInt(1844674407370955161)

	var proposals []types.CoordinatedBlockProposal

	proposalsToAdd := ocr2keepersv3.ObservationLogRecoveryProposalsLimit + ocr2keepersv3.ObservationConditionalsProposalsLimit
	for i := 0; i < proposalsToAdd; i++ {
		proposals = append(proposals, types.CoordinatedBlockProposal{
			UpkeepID: [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			Trigger: types.Trigger{
				BlockNumber: types.BlockNumber(bigNumber.Uint64()),
				BlockHash:   [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				LogTriggerExtension: &types.LogTriggerExtension{
					TxHash:      [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
					Index:       4294967295,
					BlockHash:   [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
					BlockNumber: types.BlockNumber(bigNumber.Uint64()),
				},
			},
			WorkID: "abc123fffedb8ab06d3c766b2ff1791ae277aa8efc5357729b640c432f706c99",
		})
	}

	t.Run("Add up to 100 lightly populated performables if we have capacity", func(t *testing.T) {
		observation := &ocr2keepersv3.AutomationObservation{
			UpkeepProposals: proposals,
			BlockHistory:    blockHistory,
		}

		b, _ := observation.Encode()

		results := buildResults(1000, 500)

		added, encodings := hook.addByPercentageExceeded(observation, ocr2keepersv3.ObservationPerformablesLimit, results, len(b))
		assert.Equal(t, ocr2keepersv3.ObservationPerformablesLimit, added)
		assert.Equal(t, 1, encodings)

		b, err := observation.Encode()
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(b), ocr2keepersv3.MaxObservationLength)
		assert.Equal(t, len(b), 210508)
	})

	t.Run("Add up to 100 heavily populated performables if we have capacity", func(t *testing.T) {
		observation := &ocr2keepersv3.AutomationObservation{
			UpkeepProposals: proposals,
			BlockHistory:    blockHistory,
		}

		b, _ := observation.Encode()

		results := buildResults(1000, 10000)

		added, encodings := hook.addByPercentageExceeded(observation, ocr2keepersv3.ObservationPerformablesLimit, results, len(b))
		assert.Equal(t, 65, added)
		assert.Equal(t, 2, encodings)

		b, err := observation.Encode()
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(b), ocr2keepersv3.MaxObservationLength)
		assert.Equal(t, len(b), 976528)
	})
}

func BenchmarkAddByPercentageExceded(b *testing.B) {
	results := buildResults(1000, 10000)
	var hook AddFromStagingHook
	observation := &ocr2keepersv3.AutomationObservation{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hook.addByPercentageExceeded(observation, 100, results, ocr2keepersv3.ObservationPerformablesLimit)
	}

	// ~5680891 ns/op
}

func buildResults(num, performDataSize int) []types.CheckResult {
	var res []types.CheckResult

	for i := 0; i < num; i++ {
		performData := make([]byte, performDataSize)
		for i := 0; i < performDataSize; i++ {
			performData[i] = 255
		}

		bigNumber := big.NewInt(1844674407370955161)

		res = append(res, types.CheckResult{
			PipelineExecutionState: 255,
			Retryable:              false,
			Eligible:               true,
			IneligibilityReason:    255,
			UpkeepID:               [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			Trigger: types.Trigger{
				BlockNumber: types.BlockNumber(bigNumber.Uint64()),
				BlockHash:   [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				LogTriggerExtension: &types.LogTriggerExtension{
					TxHash:      [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
					Index:       4294967295,
					BlockHash:   [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
					BlockNumber: types.BlockNumber(bigNumber.Uint64()),
				},
			},
			WorkID:       "acd4ff368edb8ab06d3c766b2ff1791ae277aa8efc5357729b640c432f706c86",
			GasAllocated: bigNumber.Uint64(),
			PerformData:  performData,
			FastGasWei:   bigNumber,
			LinkNative:   bigNumber,
		})
	}

	return res
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
