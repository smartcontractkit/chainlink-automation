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
		expectedOutcomeWorkIDs []string
	}{
		{
			name:                   "No eligible results",
			threshold:              2,
			limit:                  3,
			observations:           []ocr2keepers.AutomationObservation{},
			expectedOutcomeWorkIDs: []string{},
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
			expectedOutcomeWorkIDs: []string{},
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
			workIDs := []string{}
			for _, performable := range outcome.AgreedPerformables {
				workIDs = append(workIDs, performable.WorkID)
			}
			assert.Equal(t, tt.expectedOutcomeWorkIDs, workIDs)
		})
	}
}
