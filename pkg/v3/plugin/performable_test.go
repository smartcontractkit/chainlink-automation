package plugin

import (
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	types "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
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
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909310909090a090a09": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909320909090a090a09": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909330909090a090a09": {
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
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909310909090a090a09": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909320909090a090a09": {
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
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909310909090a090a09": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909320909090a090a09": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909330909090a090a09": {
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
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909310909090a090a09": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 1,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909320909090a090a09": {
					result: types.CheckResult{
						WorkID:     "2",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909330909090a090a09": {
					result: types.CheckResult{
						WorkID:     "3",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 3,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909340909090a090a09": {
					result: types.CheckResult{
						WorkID:     "4",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 7,
				},
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909350909090a090a09": {
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
				"000966616c73650966616c73650900090000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000000090909310909090a090a09": {
					result: types.CheckResult{
						WorkID:     "1",
						FastGasWei: big.NewInt(10),
						LinkNative: big.NewInt(10),
					},
					count: 2,
				},
				"000966616c73650966616c736509000900000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000000909093109090914090a09": {
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
