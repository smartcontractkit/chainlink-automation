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

func TestPerformables(t *testing.T) {
	tests := []struct {
		name                   string
		threshold              int
		limit                  int
		observations           []ocr2keepers.AutomationObservation
		expectedOutcomeWorkIDs []types.CheckResult
		wantResultCount        map[string]resultAndCount
	}{
		{
			name:                   "No eligible results",
			threshold:              2,
			limit:                  3,
			observations:           []ocr2keepers.AutomationObservation{},
			expectedOutcomeWorkIDs: []types.CheckResult{},
			wantResultCount:        map[string]resultAndCount{},
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
			wantResultCount: map[string]resultAndCount{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000330a0a": {
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
			wantResultCount: map[string]resultAndCount{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": {
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
			wantResultCount: map[string]resultAndCount{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000330a0a": {
					result: types.CheckResult{
						WorkID:     "3",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
			},
		},
		{
			name:      "When the count exceeds the limit, the number of results are limited with same sorting order",
			threshold: 2,
			limit:     1,
			observations: []ocr2keepers.AutomationObservation{
				{
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "4", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
						{WorkID: "5", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
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
			wantResultCount: map[string]resultAndCount{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000320a0a": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000330a0a": {
					result: types.CheckResult{
						WorkID:     "3",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000340a0a": {
					result: types.CheckResult{
						WorkID:     "4",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 7,
				},
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000350a0a": {
					result: types.CheckResult{
						WorkID:     "5",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 7,
				},
			},
		},
		{
			name:      "Duplicate work IDs with different UIDs reaching threshold",
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
						{WorkID: "1", FastGasWei: big.NewInt(10), LinkNative: big.NewInt(10)},
					},
				},
				{
					// Same workID but different fastGasWei
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(20), LinkNative: big.NewInt(10)},
					},
				},
				{
					// Same workID but different fastGasWei
					Performable: []types.CheckResult{
						{WorkID: "1", FastGasWei: big.NewInt(20), LinkNative: big.NewInt(10)},
					},
				},
			},
			expectedOutcomeWorkIDs: []types.CheckResult{
				{
					WorkID:     "1",
					FastGasWei: big.NewInt(10),
					LinkNative: big.NewInt(10),
				},
			},
			wantResultCount: map[string]resultAndCount{
				"0066616c736566616c73650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000310a0a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 2,
				},
				"0066616c736566616c7365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000031140a": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(20),
						LinkNative: big.NewInt(10),
					},
					count: 2,
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
