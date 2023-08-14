package plugin

import (
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// TODO: Add proper tests
func TestPerformables(t *testing.T) {
	tests := []struct {
		name                   string
		threshold              int
		limit                  int
		observations           []ocr2keepers.AutomationObservation
		expectedOutcomeWorkIDs []types.CheckResult
		wantResultCount        map[string]resultAndCount[types.CheckResult]
	}{
		{
			name:                   "No eligible results",
			threshold:              2,
			limit:                  3,
			observations:           []ocr2keepers.AutomationObservation{},
			expectedOutcomeWorkIDs: []types.CheckResult{},
			wantResultCount:        map[string]resultAndCount[types.CheckResult]{},
		},
		{
			name:      "No threshold met results",
			threshold: 2,
			limit:     3,
			observations: []ocr2keepers.AutomationObservation{
				{
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
			},
			expectedOutcomeWorkIDs: []types.CheckResult{},
			wantResultCount: map[string]resultAndCount[types.CheckResult]{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000330a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "3",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
			},
		},
		{
			name:      "Duplicate work IDs increase the instance count",
			threshold: 2,
			limit:     3,
			observations: []ocr2keepers.AutomationObservation{
				{
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
			},
			expectedOutcomeWorkIDs: []types.CheckResult{
				{
					WorkID:     "2",
					FastGasWei: big.NewInt(10),
					LinkNative: big.NewInt(10),
				},
			},
			wantResultCount: map[string]resultAndCount[types.CheckResult]{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
			},
		},
		{
			name:      "When the count exceeds the limit, the number of results are limited",
			threshold: 2,
			limit:     1,
			observations: []ocr2keepers.AutomationObservation{
				{
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
			},
			expectedOutcomeWorkIDs: []types.CheckResult{
				{
					WorkID:     "2",
					FastGasWei: big.NewInt(10),
					LinkNative: big.NewInt(10),
				},
			},
			wantResultCount: map[string]resultAndCount[types.CheckResult]{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000330a0a": resultAndCount[types.CheckResult]{
					result: types.CheckResult{
						WorkID:     "3",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)
			performables := newPerformables(tt.threshold, tt.limit, [16]byte{}, logger)
			for _, observation := range tt.observations {
				performables.add(observation)
			}
			outcome := ocr2keepers.AutomationOutcome{}
			performables.set(&outcome)

			assert.Equal(t, tt.expectedOutcomeWorkIDs, outcome.AgreedPerformables)
			assert.Equal(t, tt.wantResultCount, performables.resultCount)
		})
	}
}
