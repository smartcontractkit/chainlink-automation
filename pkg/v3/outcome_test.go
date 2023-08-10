package ocr2keepers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOutcomeMetadataKey(t *testing.T) {
	tests := []struct {
		key OutcomeMetadataKey
		err error
	}{
		{key: CoordinatedBlockOutcomeKey},
		{key: CoordinatedRecoveryProposalKey},
		{key: CoordinatedSamplesProposalKey},
		{
			key: "invalid key",
			err: ErrInvalidMetadataKey,
		},
	}

	for _, test := range tests {
		t.Run(string(test.key), func(t *testing.T) {
			err := ValidateOutcomeMetadataKey(test.key)

			if test.err == nil {
				assert.NoError(t, err, "no error expected")
			} else {
				assert.ErrorIs(t, err, test.err, "error should be of expected type")
			}
		})
	}
}

func TestAutomationOutcome_Encode_Decode(t *testing.T) {
	// set non-default values to test encoding/decoding
	input := AutomationOutcome{}

	expected := AutomationOutcome{}

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
		testData := AutomationOutcome{}

		ValidateAutomationOutcome(testData)
	})

	t.Run("invalid metadata key", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.ErrorIs(t, err, ErrInvalidMetadataKey, "invalid metadata key should return validation error")
	})

	t.Run("invalid check result", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.NotNil(t, err, "invalid check result should return validation error")
	})

	t.Run("invalid ring buffer", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.NotNil(t, err, "invalid ring buffer index should return validation error")
	})

	t.Run("no error on empty", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.NoError(t, err, "no error should return from empty outcome")
	})

	t.Run("no error on valid", func(t *testing.T) {
		testData := AutomationOutcome{}

		err := ValidateAutomationOutcome(testData)

		assert.NoError(t, err, "no error should return from a valid outcome")
	})
}

/*
func TestRecoveryProposals(t *testing.T) {
	tests := []struct {
		name        string
		outcome     AutomationOutcome
		expected    []ocr2keepers.CoordinatedProposal
		expectedErr error
	}{
		{
			name:        "happy path - empty",
			outcome:     AutomationOutcome{},
			expected:    nil,
			expectedErr: nil,
		},
		{
			name:    "happy path - with results",
			outcome: AutomationOutcome{},
			expected: []ocr2keepers.CoordinatedProposal{
				{UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{7})},
			},
			expectedErr: nil,
		},
		{
			name:        "error path - wrong type",
			outcome:     AutomationOutcome{},
			expected:    nil,
			expectedErr: ErrWrongDataType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prop, err := test.outcome.RecoveryProposals()

			assert.Equal(t, test.expected, prop, "proposals should match expected")

			if test.expectedErr == nil {
				assert.NoError(t, err, "no error expected")
			} else {
				assert.ErrorIs(t, err, test.expectedErr, "error should be of expected type")
			}
		})
	}
}*/

/*
func TestLatestCoordinatedBlock(t *testing.T) {
	tests := []struct {
		name        string
		outcome     AutomationOutcome
		expected    ocr2keepers.BlockKey
		expectedErr error
	}{
		{
			name:        "error path - block not available",
			outcome:     AutomationOutcome{},
			expected:    ocr2keepers.BlockKey{},
			expectedErr: ErrBlockNotAvailable,
		},
		{
			name:        "error path - block in latest wrong data type",
			outcome:     AutomationOutcome{},
			expected:    ocr2keepers.BlockKey{},
			expectedErr: ErrWrongDataType,
		},
		{
			name:        "happy path - block in history wrong data type",
			outcome:     AutomationOutcome{},
			expected:    ocr2keepers.BlockKey{},
			expectedErr: ErrWrongDataType,
		},
		{
			name:    "happy path - block in latest",
			outcome: AutomationOutcome{},
			expected: ocr2keepers.BlockKey{
				Number: 2,
			},
			expectedErr: nil,
		},
		{
			name:    "happy path - block in history",
			outcome: AutomationOutcome{},
			expected: ocr2keepers.BlockKey{
				Number: 2,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block, err := test.outcome.LatestCoordinatedBlock()

			assert.Equal(t, test.expected, block, "block key should match expected")

			if test.expectedErr == nil {
				assert.NoError(t, err, "no error expected")
			} else {
				assert.ErrorIs(t, err, test.expectedErr, "error should be of expected type")
			}
		})
	}
}


func TestSortedHistory(t *testing.T) {
	tests := []struct {
		name     string
		outcome  AutomationOutcome
		expected []BasicOutcome
	}{
		{
			name:     "happy path - no history",
			outcome:  AutomationOutcome{},
			expected: []BasicOutcome{},
		},
		{
			name: "happy path - single item",
			outcome: AutomationOutcome{
				History: []BasicOutcome{
					{
						Metadata: map[OutcomeMetadataKey]interface{}{},
					},
				},
				NextIdx: 1,
			},
			expected: []BasicOutcome{
				{
					Metadata: map[OutcomeMetadataKey]interface{}{},
				},
			},
		},
		{
			name: "happy path - ring buffer sorted latest to oldest",
			outcome: AutomationOutcome{
				History: []BasicOutcome{
					{
						Metadata: map[OutcomeMetadataKey]interface{}{
							CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
								Number: 4,
							},
						},
					},
					{
						Metadata: map[OutcomeMetadataKey]interface{}{
							CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
								Number: 5,
							},
						},
					},
					{
						Metadata: map[OutcomeMetadataKey]interface{}{
							CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
								Number: 1,
							},
						},
					},
					{
						Metadata: map[OutcomeMetadataKey]interface{}{
							CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
								Number: 2,
							},
						},
					},
					{
						Metadata: map[OutcomeMetadataKey]interface{}{
							CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
								Number: 3,
							},
						},
					},
				},
				NextIdx: 2,
			},
			expected: []BasicOutcome{
				{
					Metadata: map[OutcomeMetadataKey]interface{}{
						CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
							Number: 5,
						},
					},
				},
				{
					Metadata: map[OutcomeMetadataKey]interface{}{
						CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
							Number: 4,
						},
					},
				},
				{
					Metadata: map[OutcomeMetadataKey]interface{}{
						CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
							Number: 3,
						},
					},
				},
				{
					Metadata: map[OutcomeMetadataKey]interface{}{
						CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
							Number: 2,
						},
					},
				},
				{
					Metadata: map[OutcomeMetadataKey]interface{}{
						CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
							Number: 1,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			history := test.outcome.SortedHistory()

			assert.Equal(t, test.expected, history, "history should match expected")
		})
	}
}
*/
