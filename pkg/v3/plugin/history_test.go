package plugin

import (
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHistory(t *testing.T) {
	t.Run("nil previous outcome", func(t *testing.T) {
		next := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: nil,
				},
			},
		}

		UpdateHistory(nil, &next, 10)

		assert.Equal(t, 0, len(next.History), "history should be unchanged")
		assert.Equal(t, 0, next.NextIdx, "history should be 0, the first index to put the next history item")
	})

	t.Run("empty history in previous outcome", func(t *testing.T) {
		previous := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: nil,
				},
			},
		}

		next := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: nil,
				},
			},
		}

		assert.Equal(t, 0, len(next.History), "history should be empty before the update")

		UpdateHistory(&previous, &next, 10)

		assert.Equal(t, 1, len(next.History), "history should be updated to 1")
		assert.Equal(t, 1, next.NextIdx, "history should be 1, the next index to put the next history item")
	})

	t.Run("history of length 1 in previous outcome", func(t *testing.T) {
		previous := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
				},
			},
			History: []ocr2keepersv3.BasicOutcome{
				{
					Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
						ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("1"),
					},
				},
			},
			NextIdx: 1,
		}

		next := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: nil,
				},
			},
		}

		expected := []ocr2keepersv3.BasicOutcome{
			{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("1"),
				},
			},
			{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
				},
			},
		}

		assert.Equal(t, 0, len(next.History), "history should be empty before the update")

		UpdateHistory(&previous, &next, 10)

		assert.Equal(t, 2, len(next.History), "history should be updated to 2")
		assert.Equal(t, 2, next.NextIdx, "history should be 2, the next index to put the next history item")
		assert.Equal(t, expected, next.History, "history items should match expected order")
	})

	t.Run("history of max length in previous outcome", func(t *testing.T) {
		previous := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("4"),
				},
			},
			History: []ocr2keepersv3.BasicOutcome{
				{
					Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
						ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("1"),
					},
				},
				{
					Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
						ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
					},
				},
				{
					Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
						ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("3"),
					},
				},
			},
			NextIdx: 0,
		}

		next := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: nil,
				},
			},
		}

		expected := []ocr2keepersv3.BasicOutcome{
			{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("4"),
				},
			},
			{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("2"),
				},
			},
			{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey("3"),
				},
			},
		}

		assert.Equal(t, 0, len(next.History), "history should be empty before the update")

		UpdateHistory(&previous, &next, 3)

		assert.Equal(t, 3, len(next.History), "history should be maximum length")
		assert.Equal(t, 1, next.NextIdx, "history should be 1, the next index to put the next history item")
		assert.Equal(t, expected, next.History, "history items should match expected order")
	})
}
