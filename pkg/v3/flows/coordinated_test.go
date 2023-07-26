package flows

import (
	"testing"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestHistoryFromRingBuffer(t *testing.T) {
	ring := []ocr2keepersv3.BasicOutcome{
		{
			Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
				"key": 3,
			},
		},
		{
			Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
				"key": 1,
			},
		},
		{
			Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
				"key": 2,
			},
		},
	}

	nextIdx := 1

	values := historyFromRingBuffer(ring, nextIdx)

	var sorted []int
	for _, v := range values {
		sorted = append(sorted, v.Metadata["key"].(int))
	}

	assert.Equal(t, []int{1, 2, 3}, sorted, "sort order should be oldest to newest")
}
