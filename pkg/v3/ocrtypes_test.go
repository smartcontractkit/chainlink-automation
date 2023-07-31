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
						Extension: struct {
							Hash  string
							Value int64
						}{
							Hash:  "0xhash",
							Value: 18,
						},
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
						Extension:   []byte(`{"Hash":"0xhash","Value":18}`),
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

func TestValidateAutomationObservation(t *testing.T) {
	t.Run("invalid instructions", func(t *testing.T) {
		testData := AutomationObservation{
			Instructions: []instructions.Instruction{
				"invalid instruction",
			},
		}

		err := ValidateAutomationObservation(testData)

		assert.ErrorIs(t, err, instructions.ErrInvalidInstruction, "invalid instruction should return validation error")
	})

	t.Run("invalid metadata key", func(t *testing.T) {
		testData := AutomationObservation{
			Metadata: map[ObservationMetadataKey]interface{}{
				"invalid key": "string",
			},
		}

		err := ValidateAutomationObservation(testData)

		assert.ErrorIs(t, err, ErrInvalidMetadataKey, "invalid metadata key should return validation error")
	})

	t.Run("invalid check result", func(t *testing.T) {
		testData := AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{},
			},
		}

		err := ValidateAutomationObservation(testData)

		assert.NotNil(t, err, "invalid check result should return validation error")
	})

	t.Run("no error on empty", func(t *testing.T) {
		testData := AutomationObservation{}

		err := ValidateAutomationObservation(testData)

		assert.NoError(t, err, "no error should return from empty observation")
	})

	t.Run("no error on valid", func(t *testing.T) {
		testData := AutomationObservation{
			Instructions: []instructions.Instruction{
				instructions.DoCoordinateBlock,
			},
			Metadata: map[ObservationMetadataKey]interface{}{
				BlockHistoryObservationKey: ocr2keepers.BlockKey("3"),
			},
			Performable: []ocr2keepers.CheckResult{
				{
					Eligible:     true,
					Retryable:    false,
					GasAllocated: 1,
					Payload: ocr2keepers.UpkeepPayload{
						ID: "test",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID: ocr2keepers.UpkeepIdentifier("test"),
						},
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 10,
							BlockHash:   "0x",
						},
					},
				},
			},
		}

		err := ValidateAutomationObservation(testData)

		assert.NoError(t, err, "no error should return from a valid observation")
	})
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

func TestValidateAutomationOutcome(t *testing.T) {
	t.Run("invalid instructions", func(t *testing.T) {
		testData := AutomationOutcome{
			Instructions: []instructions.Instruction{
				"invalid instruction",
			},
		}

		err := ValidateAutomationOutcome(testData)

		assert.ErrorIs(t, err, instructions.ErrInvalidInstruction, "invalid instruction should return validation error")
	})

	t.Run("invalid metadata key", func(t *testing.T) {
		testData := AutomationOutcome{
			BasicOutcome: BasicOutcome{
				Metadata: map[OutcomeMetadataKey]interface{}{
					"invalid key": "string",
				},
			},
		}

		err := ValidateAutomationOutcome(testData)

		assert.ErrorIs(t, err, ErrInvalidMetadataKey, "invalid metadata key should return validation error")
	})

	t.Run("invalid check result", func(t *testing.T) {
		testData := AutomationOutcome{
			BasicOutcome: BasicOutcome{
				Performable: []ocr2keepers.CheckResult{
					{},
				},
			},
		}

		err := ValidateAutomationOutcome(testData)

		assert.NotNil(t, err, "invalid check result should return validation error")
	})

	t.Run("invalid ring buffer", func(t *testing.T) {
		testData := AutomationOutcome{
			History: []BasicOutcome{
				{},
			},
			NextIdx: 3,
		}

		err := ValidateAutomationOutcome(testData)

		assert.NotNil(t, err, "invalid ring buffer index should return validation error")
	})

	t.Run("no error on empty", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.NoError(t, err, "no error should return from empty outcome")
	})

	t.Run("no error on valid", func(t *testing.T) {
		testData := AutomationOutcome{
			Instructions: []instructions.Instruction{
				instructions.DoCoordinateBlock,
			},
			BasicOutcome: BasicOutcome{
				Metadata: map[OutcomeMetadataKey]interface{}{
					CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("3"),
				},
				Performable: []ocr2keepers.CheckResult{
					{
						Eligible:     true,
						Retryable:    false,
						GasAllocated: 1,
						Payload: ocr2keepers.UpkeepPayload{
							ID: "test",
							Upkeep: ocr2keepers.ConfiguredUpkeep{
								ID: ocr2keepers.UpkeepIdentifier("test"),
							},
							Trigger: ocr2keepers.Trigger{
								BlockNumber: 10,
								BlockHash:   "0x",
							},
						},
					},
				},
			},
		}

		err := ValidateAutomationOutcome(testData)

		assert.NoError(t, err, "no error should return from a valid outcome")
	})

}
