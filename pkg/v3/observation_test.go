package ocr2keepers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestAutomationObservation(t *testing.T) {
	// set non-default values to test encoding/decoding
	input := AutomationObservation{
		Performable: []ocr2keepers.CheckResult{
			{
				UpkeepID:    [32]byte{111},
				Retryable:   true,
				Eligible:    true,
				PerformData: []byte("testing"),
			},
		},
	}

	expected := AutomationObservation{
		Performable: []ocr2keepers.CheckResult{
			{
				UpkeepID:    [32]byte{111},
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

func TestValidateAutomationObservation(t *testing.T) {
	t.Run("invalid check result", func(t *testing.T) {
		testData := AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{},
			},
		}

		err := ValidateAutomationObservation(testData, mockGenerateWorkID)

		assert.NotNil(t, err, "invalid check result should return validation error")
	})

	t.Run("no error on empty", func(t *testing.T) {
		testData := AutomationObservation{}

		err := ValidateAutomationObservation(testData, mockGenerateWorkID)

		assert.NoError(t, err, "no error should return from empty observation")
	})

	t.Run("no error on valid", func(t *testing.T) {
		testData := AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{
					Eligible:     true,
					Retryable:    false,
					GasAllocated: 1,
					UpkeepID:     ocr2keepers.UpkeepIdentifier([32]byte{123}),
				},
			},
		}

		err := ValidateAutomationObservation(testData, mockGenerateWorkID)

		assert.NoError(t, err, "no error should return from a valid observation")
	})
}

func mockGenerateWorkID(id types.UpkeepIdentifier, trigger types.Trigger) string {
	wid := string(id[:])
	if trigger.LogTriggerExtension != nil {
		wid += string(trigger.LogTriggerExtension.LogIdentifier())
	}
	return wid
}
