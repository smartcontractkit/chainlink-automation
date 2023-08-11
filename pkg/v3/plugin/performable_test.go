package plugin

import (
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/stretchr/testify/assert"
)

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
						{WorkID: "1"},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2"},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "3"},
					},
				},
			},
			expectedOutcomeWorkIDs: []string{},
		},
		{
			name:      "Threshold met results",
			threshold: 2,
			limit:     3,
			observations: []ocr2keepers.AutomationObservation{
				{
					Performable: []types.CheckResult{
						{WorkID: "1"},
						{WorkID: "2"},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "1"},
					},
				},
				{
					Performable: []types.CheckResult{
						{WorkID: "2"},
					},
				},
			},
			expectedOutcomeWorkIDs: []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			performables := newPerformables(tt.threshold, tt.limit, [16]byte{})
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
