package ocr2keepers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
)

func TestAutomationObservation(t *testing.T) {
	// set non-default values to test encoding/decoding
	input := AutomationObservation{
		Instructions: []instructions.Instruction{"instruction1", "instruction2"},
		Metadata: map[ObservationMetadataKey]interface{}{
			BlockHistoryObservationKey: ocr2keepers.BlockHistory([]ocr2keepers.BlockKey{("2")}),
		},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
					Upkeep: ocr2keepers.ConfiguredUpkeep{
						ID:     []byte("111"),
						Type:   1,
						Config: "value",
					},
					CheckData: []byte("check data"),
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 4,
						BlockHash:   "hash",
						Extension:   8,
					},
				},
				Retryable:   true,
				Eligible:    true,
				PerformData: []byte("testing"),
			},
		},
	}

	expected := AutomationObservation{
		Instructions: []instructions.Instruction{"instruction1", "instruction2"},
		Metadata: map[ObservationMetadataKey]interface{}{
			BlockHistoryObservationKey: ocr2keepers.BlockHistory([]ocr2keepers.BlockKey{("2")}),
		},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
					Upkeep: ocr2keepers.ConfiguredUpkeep{
						ID:     []byte("111"),
						Type:   1,
						Config: []byte(`"value"`),
					},
					CheckData: []byte("check data"),
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 4,
						BlockHash:   "hash",
						Extension:   []byte("8"),
					},
				},
				Retryable:   true,
				Eligible:    true,
				PerformData: []byte("testing"),
			},
		},
	}

	jsonData, _ := json.Marshal(input)
	data, err := input.Encode()

	assert.Equal(t, jsonData, data, "json marshalling should return the same result")
	assert.NoError(t, err, "no error from encoding")

	result, err := DecodeAutomationObservation(data)
	assert.NoError(t, err, "no error from decoding")

	assert.Equal(t, expected, result, "final result from encoding and decoding should match")
}

func TestAutomationOutcome(t *testing.T) {
	// set non-default values to test encoding/decoding
	input := AutomationOutcome{
		BasicOutcome: BasicOutcome{
			Metadata: map[OutcomeMetadataKey]interface{}{
				CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
			},
			Performable: []ocr2keepers.CheckResult{
				{
					Payload: ocr2keepers.UpkeepPayload{
						ID: "abc",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID:     []byte("111"),
							Type:   1,
							Config: "value",
						},
						CheckData: []byte("check data"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   "hash",
							Extension:   8,
						},
					},
					Retryable:   true,
					Eligible:    true,
					PerformData: []byte("testing"),
				},
			},
		},
		Instructions: []instructions.Instruction{"instruction1", "instruction2"},
	}

	expected := AutomationOutcome{
		BasicOutcome: BasicOutcome{
			Metadata: map[OutcomeMetadataKey]interface{}{
				CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
			},
			Performable: []ocr2keepers.CheckResult{
				{
					Payload: ocr2keepers.UpkeepPayload{
						ID: "abc",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID:     []byte("111"),
							Type:   1,
							Config: []byte(`"value"`),
						},
						CheckData: []byte("check data"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   "hash",
							Extension:   []byte("8"),
						},
					},
					Retryable:   true,
					Eligible:    true,
					PerformData: []byte("testing"),
				},
			},
		},
		Instructions: []instructions.Instruction{"instruction1", "instruction2"},
	}

	jsonData, _ := json.Marshal(input)
	data, err := input.Encode()

	assert.Equal(t, jsonData, data, "json marshalling should return the same result")
	assert.NoError(t, err, "no error from encoding")

	result, err := DecodeAutomationOutcome(data)
	assert.NoError(t, err, "no error from decoding")

	assert.Equal(t, expected, result, "final result from encoding and decoding should match")

}
