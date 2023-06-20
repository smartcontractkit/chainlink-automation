package ocr2keepers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
)

func TestAutomationObservation(t *testing.T) {
	// set non-default values to test encoding/decoding
	expected := AutomationObservation{
		Instructions: []instruction.Instruction{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
					Upkeep: ocr2keepers.ConfiguredUpkeep{
						ID:   []byte("111"),
						Type: 1,
					},
					CheckData: []byte("check data"),
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 4,
						BlockHash:   "hash",
					},
				},
				Retryable:   true,
				Eligible:    true,
				PerformData: []byte("testing"),
			},
		},
	}

	jsonData, _ := json.Marshal(expected)
	data, err := expected.Encode()

	assert.Equal(t, jsonData, data, "json marshalling should return the same result")
	assert.NoError(t, err, "no error from encoding")

	result, err := DecodeAutomationObservation(data)
	assert.NoError(t, err, "no error from decoding")

	assert.Equal(t, expected, result, "final result from encoding and decoding should match")
}

func TestAutomationOutcome(t *testing.T) {
	// set non-default values to test encoding/decoding
	expected := AutomationOutcome{
		Instructions: []instruction.Instruction{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
					Upkeep: ocr2keepers.ConfiguredUpkeep{
						ID:   []byte("111"),
						Type: 1,
					},
					CheckData: []byte("check data"),
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 4,
						BlockHash:   "hash",
					},
				},
				Retryable:   true,
				Eligible:    true,
				PerformData: []byte("testing"),
			},
		},
	}

	jsonData, _ := json.Marshal(expected)
	data, err := expected.Encode()

	assert.Equal(t, jsonData, data, "json marshalling should return the same result")
	assert.NoError(t, err, "no error from encoding")

	result, err := DecodeAutomationOutcome(data)
	assert.NoError(t, err, "no error from decoding")

	assert.Equal(t, expected, result, "final result from encoding and decoding should match")

}
